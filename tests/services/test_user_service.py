import pytest
from uuid import UUID
from sqlalchemy.ext.asyncio import AsyncSession
from sqlmodel import SQLModel

from july.db.models import User, UserIdentifier, IdentifierType
from july.schema import EmailAddress, OAuthUser, OAuthProvider

from july.services.user_service import UserService


@pytest.fixture
def user_service(db_session: AsyncSession) -> UserService:
    return UserService(db_session)


@pytest.fixture
def sample_emails() -> list[EmailAddress]:
    return [
        EmailAddress(email="primary@example.com", verified=True, primary=True),
        EmailAddress(email="secondary@example.com", verified=True, primary=False),
        EmailAddress(email="unverified@example.com", verified=False, primary=False),
    ]


class TestFindByIdentifier:
    async def test_find_existing_identifier(
        self, db_session: AsyncSession, user_service: UserService, user: User
    ):
        identifier = UserIdentifier(
            value="github:12345",
            type=IdentifierType.GITHUB,
            user_id=user.id,
            verified=True,
        )
        db_session.add(identifier)
        await db_session.commit()

        found = await user_service.find_by_identifier(IdentifierType.GITHUB, "12345")

        assert found is not None
        assert found.id == user.id

    async def test_find_nonexistent_identifier(self, user_service: UserService):
        found = await user_service.find_by_identifier(
            IdentifierType.GITHUB, "nonexistent"
        )
        assert found is None


class TestFindByEmail:
    async def test_find_existing_email(
        self, db_session: AsyncSession, user_service: UserService, user: User
    ):
        identifier = UserIdentifier(
            value="email:test@example.com",
            type=IdentifierType.EMAIL,
            user_id=user.id,
            verified=True,
        )
        db_session.add(identifier)
        await db_session.commit()

        email = EmailAddress(email="test@example.com", verified=True, primary=True)
        found = await user_service.find_by_email(email)

        assert found is not None
        assert found.id == user.id

    async def test_find_nonexistent_email(self, user_service: UserService):
        email = EmailAddress(email="nobody@example.com", verified=True, primary=True)
        found = await user_service.find_by_email(email)
        assert found is None


class TestFindByOAuth:
    async def test_find_github_user(
        self, db_session: AsyncSession, user_service: UserService, user: User
    ):
        identifier = UserIdentifier(
            value="github:67890",
            type=IdentifierType.GITHUB,
            user_id=user.id,
            verified=True,
        )
        db_session.add(identifier)
        await db_session.commit()

        found = await user_service.find_by_oauth("github", "67890")

        assert found is not None
        assert found.id == user.id

    async def test_find_gitlab_user(
        self, db_session: AsyncSession, user_service: UserService, user: User
    ):
        identifier = UserIdentifier(
            value="gitlab:11111",
            type=IdentifierType.GITLAB,
            user_id=user.id,
            verified=True,
        )
        db_session.add(identifier)
        await db_session.commit()

        found = await user_service.find_by_oauth("gitlab", "11111")

        assert found is not None
        assert found.id == user.id


class TestOAuthLoginOrRegister:
    async def test_new_user_registration(
        self, db_session: AsyncSession, user_service: UserService
    ):
        emails = [EmailAddress(email="new@example.com", verified=True, primary=True)]

        user, is_new = await user_service.oauth_login_or_register(
            OAuthUser(
                username="ted",
                provider=OAuthProvider.GITHUB,
                id="new-user-123",
                name="New User",
                emails=emails,
                avatar_url="https://example.com/avatar.png",
                data={"test": "meta"},
            )
        )

        assert is_new is True
        assert user.name == "New User"
        assert user.avatar_url == "https://example.com/avatar.png"

        # Verify OAuth identifier was created
        found = await user_service.find_by_oauth("github", "new-user-123")
        assert found is not None
        assert found.id == user.id

        # Verify email was linked
        found_by_email = await user_service.find_by_email(emails[0])
        assert found_by_email is not None
        assert found_by_email.id == user.id

    async def test_existing_oauth_user_login(
        self, db_session: AsyncSession, user_service: UserService, user: User
    ):
        # Set up existing OAuth link
        identifier = UserIdentifier(
            value="github:existing-123",
            type=IdentifierType.GITHUB,
            user_id=user.id,
            verified=True,
        )
        db_session.add(identifier)
        await db_session.commit()

        emails = [
            EmailAddress(email="new-email@example.com", verified=True, primary=True)
        ]

        found_user, is_new = await user_service.oauth_login_or_register(
            OAuthUser(
                username="ted",
                provider=OAuthProvider.GITHUB,
                id="existing-123",
                name="Updated Name",
                emails=emails,
                avatar_url="https://example.com/new-avatar.png",
                data={"test": "meta"},
            )
        )

        assert is_new is True  # Not a new user, but email was added
        assert found_user.id == user.id
        assert found_user.name == "Updated Name"
        assert found_user.avatar_url == "https://example.com/new-avatar.png"

    async def test_link_oauth_to_existing_email_user(
        self, db_session: AsyncSession, user_service: UserService, user: User
    ):
        # Set up existing email
        identifier = UserIdentifier(
            value="email:existing@example.com",
            type=IdentifierType.EMAIL,
            user_id=user.id,
            verified=True,
        )
        db_session.add(identifier)
        await db_session.commit()

        emails = [
            EmailAddress(email="existing@example.com", verified=True, primary=True)
        ]

        found_user, is_new = await user_service.oauth_login_or_register(
            OAuthUser(
                username="ted",
                provider=OAuthProvider.GITHUB,
                id="new-oauth-456",
                name="Same User",
                emails=emails,
                avatar_url=None,
                data={"test": "meta"},
            )
        )

        assert is_new is False
        assert found_user.id == user.id

        # Verify OAuth was linked
        oauth_user = await user_service.find_by_oauth("github", "new-oauth-456")
        assert oauth_user is not None
        assert oauth_user.id == user.id

    async def test_multiple_emails_find_same_user(
        self, db_session: AsyncSession, user_service: UserService, user: User
    ):
        # Set up multiple emails for same user
        for email in ["one@example.com", "two@example.com"]:
            identifier = UserIdentifier(
                value=f"email:{email}",
                type=IdentifierType.EMAIL,
                user_id=user.id,
                verified=True,
            )
            db_session.add(identifier)
        await db_session.commit()

        emails = [
            EmailAddress(email="one@example.com", verified=True, primary=True),
            EmailAddress(email="two@example.com", verified=True, primary=False),
        ]

        found_user, is_new = await user_service.oauth_login_or_register(
            OAuthUser(
                username="ted",
                provider=OAuthProvider.GITLAB,
                id="gitlab-789",
                name="Same User",
                emails=emails,
                avatar_url=None,
                data={"test": "meta"},
            )
        )

        assert is_new is False
        assert found_user.id == user.id

    async def test_multiple_emails_different_users_raises(
        self, db_session: AsyncSession, user_service: UserService, user: User
    ):
        other_user = User(name="Other User", username="ted2")
        db_session.add(other_user)
        await db_session.commit()

        # Email owned by first user
        db_session.add(
            UserIdentifier(
                value="email:user1@example.com",
                type=IdentifierType.EMAIL,
                user_id=user.id,
                verified=True,
            )
        )
        # Email owned by second user
        db_session.add(
            UserIdentifier(
                value="email:user2@example.com",
                type=IdentifierType.EMAIL,
                user_id=other_user.id,
                verified=True,
            )
        )
        await db_session.commit()

        emails = [
            EmailAddress(email="user1@example.com", verified=True, primary=True),
            EmailAddress(email="user2@example.com", verified=True, primary=False),
        ]

        with pytest.raises(ValueError, match="multiple existing users"):
            await user_service.oauth_login_or_register(
                OAuthUser(
                    username="ted",
                    provider=OAuthProvider.GITHUB,
                    id="conflict-user",
                    name="Conflict User",
                    emails=emails,
                    avatar_url=None,
                    data={"test": "meta"},
                )
            )

    async def test_only_verified_emails_used(
        self, db_session: AsyncSession, user_service: UserService
    ):
        emails = [
            EmailAddress(email="unverified@example.com", verified=False, primary=True),
        ]

        user, is_new = await user_service.oauth_login_or_register(
            OAuthUser(
                username="ted",
                provider=OAuthProvider.GITHUB,
                id="no-verified-email",
                name="No Email User",
                emails=emails,
                avatar_url=None,
                data={"test": "meta"},
            )
        )

        assert is_new is True

        # Unverified email should not be linked
        found = await user_service.find_by_email(emails[0])
        assert found is None
