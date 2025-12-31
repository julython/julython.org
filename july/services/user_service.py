from typing import Any
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.dialects.postgresql import insert
from sqlmodel import col
from structlog.stdlib import get_logger

from july.db.models import User, UserIdentifier
from july.schema import EmailAddress, IdentifierType, OAuthUser
from july.utils import times

logger = get_logger(__name__)


class UserService:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def find_by_key(self, key: str) -> User | None:
        stmt = (
            select(User)
            .join(UserIdentifier, col(User.id) == col(UserIdentifier.user_id))
            .where(col(UserIdentifier.value) == key)
        )
        result = await self.session.execute(stmt)
        return result.scalar()

    async def find_by_identifier(self, type: IdentifierType, value: str) -> User | None:
        key = f"{type.value}:{value}"
        stmt = (
            select(User)
            .join(UserIdentifier, col(User.id) == col(UserIdentifier.user_id))
            .where(col(UserIdentifier.value) == key)
        )
        result = await self.session.execute(stmt)
        return result.scalar()

    async def find_by_email(self, email: EmailAddress) -> User | None:
        return await self.find_by_identifier(IdentifierType.EMAIL, email.email)

    async def find_by_oauth(self, provider: str, provider_user_id: str) -> User | None:
        return await self.find_by_identifier(IdentifierType(provider), provider_user_id)

    async def upsert_identifier(
        self,
        user: User,
        type: IdentifierType,
        value: str,
        data: dict[str, Any],
        verified: bool = False,
        primary: bool = False,
    ) -> tuple[UserIdentifier, bool]:
        """
        Insert or update an identifier.

        Raises ValueError if the identifier exists but belongs to a different user.

        Returns: tuple of the identifier and bool created
        """
        key = f"{type.value}:{value}"
        now = times.now()

        values = {
            "value": key,
            "type": type,
            "user_id": user.id,
            "verified": verified,
            "primary": primary,
            "created_at": now,
            "updated_at": now,
            "data": data,
        }

        update_set = {
            "verified": verified,
            "primary": primary,
            "updated_at": now,
            "data": data,
        }

        stmt = (
            insert(UserIdentifier)
            .values(**values)
            .on_conflict_do_update(
                index_elements=["value"],
                set_=update_set,
                where=col(UserIdentifier.user_id) == user.id,
            )
            .returning(UserIdentifier)
        )

        result = await self.session.execute(stmt)
        identifier = result.scalar_one_or_none()

        if identifier is None:
            raise ValueError(f"This {type.value} identifier is linked to another user")

        created = identifier.created_at == identifier.updated_at
        return identifier, created

    async def oauth_login_or_register(self, oauth_user: OAuthUser) -> tuple[User, bool]:
        """
        Returns (user, is_new_user).

        1. Find by OAuth provider ID -> login, update data
        2. Find by email -> link account
        3. Create new user
        """
        identifier_type = IdentifierType(oauth_user.provider)
        verified_emails = [e for e in oauth_user.emails if e.verified]

        # whether the user or an identity was created
        created = False

        # 1. Existing OAuth User
        if user := await self.find_by_key(oauth_user.key):
            user.name = oauth_user.name or user.name
            user.avatar_url = oauth_user.avatar_url or user.avatar_url
            self.session.add(user)

            await self.upsert_identifier(
                user,
                identifier_type,
                oauth_user.id,
                verified=True,
                data=oauth_user.data,
            )
            created = await self._upsert_emails(user, verified_emails)
            await self.session.commit()
            return user, created

        # 2. Check if any emails match existing users
        existing_user = await self._find_user_by_emails(verified_emails)

        if existing_user is not None:
            await self.upsert_identifier(
                existing_user,
                identifier_type,
                oauth_user.id,
                verified=True,
                data=oauth_user.data,
            )
            created = await self._upsert_emails(existing_user, verified_emails)
            await self.session.commit()
            return existing_user, created

        # 3. New user
        new_user = User(
            name=oauth_user.name or "",
            username=oauth_user.username,
            avatar_url=oauth_user.avatar_url,
        )
        self.session.add(new_user)
        await self.session.flush()

        await self.upsert_identifier(
            new_user,
            identifier_type,
            oauth_user.id,
            verified=True,
            data=oauth_user.data,
        )
        await self._upsert_emails(new_user, verified_emails)
        await self.session.commit()
        return new_user, True

    async def _find_user_by_emails(self, emails: list[EmailAddress]) -> User | None:
        existing_user: User | None = None

        for email in emails:
            if found := await self.find_by_email(email):
                if existing_user is None:
                    existing_user = found
                elif existing_user.id != found.id:
                    raise ValueError(
                        "Found multiple existing users with the same emails as this new user!"
                    )

        return existing_user

    async def _upsert_emails(self, user: User, emails: list[EmailAddress]) -> bool:
        emails_added = False
        for email in emails:
            _, created = await self.upsert_identifier(
                user,
                IdentifierType.EMAIL,
                value=email.email,
                primary=email.primary,
                verified=email.verified,
                data={},
            )
            emails_added = emails_added or created
        return emails_added
