from typing import Any, Optional
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlmodel import col
from structlog.stdlib import get_logger

from july.db.models import User, UserIdentifier, IdentifierType
from july.schema import EmailAddress

logger = get_logger(__name__)


class UserService:
    def __init__(self, session: AsyncSession):
        self.session = session

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

    def add_identifier(
        self,
        user: User,
        type: IdentifierType,
        value: str,
        verified: bool = False,
        primary: bool = False,
        metadata: Optional[dict[str, Any]] = None,
    ) -> UserIdentifier:
        key = f"{type.value}:{value}"
        identifier = UserIdentifier(
            user_id=user.id,
            value=key,
            type=type,
            verified=verified,
            primary=primary,
            value_metadata=metadata,
        )
        self.session.add(identifier)
        return identifier

    async def oauth_login_or_register(
        self,
        provider: str,
        provider_user_id: str,
        name: str,
        emails: list[EmailAddress],
        avatar_url: str | None,
    ) -> tuple[User, bool]:
        """
        Returns (user, is_new_user).

        1. Find by OAuth provider ID -> login
        2. Find by email -> link account
        3. Create new user
        """

        verified_emails = [e for e in emails if e.verified]

        # Whether a user or email was created
        created: bool = False

        # Existing OAuth User, make sure all emails linked
        if user := await self.find_by_oauth(provider, provider_user_id):
            logger.info(f"Updating existing user: {user.name}", emails=verified_emails)
            user.name = name or user.name
            user.avatar_url = avatar_url or user.avatar_url
            self.session.add(user)
            created = await self.link_emails(user, verified_emails)
            await self.session.commit()
            return user, created

        exisiting_user: Optional[User] = None

        # Check if any emails already are owned by a user
        for email in verified_emails:
            if exisiting := await self.find_by_email(email):
                if exisiting_user is None:
                    exisiting_user = exisiting
                elif exisiting_user.id != exisiting.id:
                    # oh boy we got a problem here bail!
                    logger.error(f"oh joy emails be conflict: {verified_emails}")
                    raise ValueError(
                        "Found multiple exisiting users with the same emails as this new user!"
                    )

        if exisiting_user is not None:
            await self.link_provider(exisiting_user, provider, provider_user_id)
            created = await self.link_emails(exisiting_user, verified_emails)
            await self.session.commit()
            return exisiting_user, created

        # New user
        new_user = User(name=name, avatar_url=avatar_url)
        logger.info(f"Yay new user: {new_user.name}", emails=verified_emails)
        self.session.add(new_user)
        await self.link_provider(new_user, provider, provider_user_id)
        await self.link_emails(new_user, verified_emails)
        await self.session.commit()
        return new_user, True

    async def link_emails(self, user: User, emails: list[EmailAddress]) -> bool:
        """Link emails to user.

        Returns:
            Boolean whether any email was added
        """
        linked = [await self._link_email(user, email) for email in emails]
        return any(linked)

    async def _link_email(self, user: User, email: EmailAddress) -> bool:
        """Link email to user.

        Returns:
            Boolean whether email was added or not
        """
        if existing := await self.find_by_email(email):
            if existing.id != user.id:
                raise ValueError(f"This email is linked to another user")
            return False  # Already linked
        self.add_identifier(
            user,
            IdentifierType.EMAIL,
            value=email.email,
            primary=email.primary,
            verified=email.verified,
        )
        return True

    async def link_provider(
        self,
        user: User,
        provider: str,
        provider_user_id: str,
    ) -> None:
        """Link an OAuth provider to existing user."""
        identifier_type = IdentifierType(provider)

        if existing := await self.find_by_oauth(provider, provider_user_id):
            if existing.id != user.id:
                raise ValueError(f"This {provider} account is linked to another user")
            return  # Already linked

        self.add_identifier(user, identifier_type, provider_user_id, verified=True)
