from fastapi import HTTPException
from sqlalchemy import select
from sqlmodel import col
import structlog
import uuid
import pytest
from datetime import datetime, timezone, timedelta

from july.services.game_service import GameService, claim_orphan_commits_task
from july.services.user_service import UserService
from july.db.models import (
    AnalysisStatus,
    Board,
    Game,
    Language,
    LanguageBoard,
    Player,
    Project,
    User,
    IdentifierType,
)
from july.utils import times

from tests.fixtures import INAUGURAL_GAME

LOG = structlog.stdlib.get_logger(__name__)


@pytest.fixture()
def game_service(db_session) -> GameService:
    return GameService(db_session)


@pytest.fixture()
def user_service(db_session) -> UserService:
    return UserService(db_session)


async def _create_user(db_session, username: str):
    user = User(
        id=uuid.uuid7(),  # type: ignore
        name=username,
        username=username,
    )
    db_session.add(user)
    await db_session.flush()
    return user


class TestGetActiveGame:

    async def test_returns_active_game(
        self, active_game: Game, game_service: GameService
    ):
        game = await game_service.get_active_game(INAUGURAL_GAME)
        assert game is not None
        assert game.id == active_game.id

    async def test_returns_none_when_no_active_game(self, game_service: GameService):
        game = await game_service.get_active_game(INAUGURAL_GAME)
        assert game is None

    @pytest.mark.parametrize(
        "offset_days,expected_found",
        [
            pytest.param(0, True, id="during"),
            pytest.param(15, True, id="mid-game"),
            pytest.param(-1, False, id="before-start"),
            pytest.param(32, False, id="after-end"),
            pytest.param(30, True, id="last-day"),
        ],
    )
    async def test_timing_boundaries(
        self,
        active_game: Game,
        game_service: GameService,
        offset_days: int,
        expected_found: bool,
    ):
        check_time = datetime(2012, 7, 1, tzinfo=timezone.utc) + timedelta(
            days=offset_days
        )
        game = await game_service.get_active_game(check_time)

        if expected_found:
            assert game is not None
            assert game.id == active_game.id
        else:
            assert game is None


class TestGetActiveOrLatestGame:

    async def test_returns_active_game_when_exists(
        self, active_game: Game, game_service: GameService
    ):
        game = await game_service.get_active_or_latest_game(INAUGURAL_GAME)
        assert game is not None
        assert game.id == active_game.id

    async def test_returns_latest_completed_game(
        self, game_service: GameService, db_session
    ):
        old_game = await game_service.create_julython_game(2011, is_active=False)
        newer_game = await game_service.create_julython_game(2012, is_active=False)

        after_both = datetime(2013, 1, 1, tzinfo=timezone.utc)
        game = await game_service.get_active_or_latest_game(after_both)

        assert game is not None
        assert game.id == newer_game.id

    async def test_returns_none_when_no_games(self, game_service: GameService):
        with pytest.raises(HTTPException):
            await game_service.get_active_or_latest_game()


class TestCreateGame:

    async def test_deactivates_others(
        self, active_game: Game, game_service: GameService
    ):
        new_game = await game_service.create_julython_game(
            year=2013, is_active=True, deactivate_others=True
        )
        assert new_game is not None

        old_game = await game_service.get_active_game(INAUGURAL_GAME)
        assert old_game is None

    async def test_keeps_others_active_when_not_deactivating(
        self, active_game: Game, game_service: GameService
    ):
        await game_service.create_julython_game(
            year=2013, is_active=True, deactivate_others=False
        )

        old_game = await game_service.get_active_game(INAUGURAL_GAME)
        assert old_game is not None
        assert old_game.id == active_game.id

    @pytest.mark.parametrize(
        "year,month,expected_name",
        [
            pytest.param(2024, 7, "Julython 2024", id="july-2024"),
            pytest.param(2024, 1, "J(an)ulython 2024", id="jan-2024"),
            pytest.param(2025, 7, "Julython 2025", id="july-2025"),
            pytest.param(2023, 1, "J(an)ulython 2023", id="jan-2023"),
            pytest.param(2023, 9, "Testathon Fall 2023", id="fall-2023"),
            pytest.param(2023, 4, "Testathon Spring 2023", id="spring-2023"),
        ],
    )
    async def test_julython_naming(
        self,
        game_service: GameService,
        year: int,
        month: int,
        expected_name: str,
    ):
        game = await game_service.create_julython_game(year=year, month=month)
        assert game.name == expected_name

    @pytest.mark.parametrize(
        "month",
        [
            pytest.param(-6, id="negative"),
            pytest.param(14, id="invalid"),
        ],
    )
    async def test_invalid_month_raises(self, game_service: GameService, month: int):
        with pytest.raises(ValueError, match=f"Invalid month {month}"):
            await game_service.create_julython_game(year=2024, month=month)

    @pytest.mark.parametrize(
        "commit_points,project_points",
        [
            pytest.param(1, 10, id="default"),
            pytest.param(2, 20, id="double"),
            pytest.param(5, 50, id="5x"),
            pytest.param(0, 100, id="commits-zero"),
        ],
    )
    async def test_custom_points(
        self,
        game_service: GameService,
        commit_points: int,
        project_points: int,
    ):
        game = await game_service.create_game(
            name="Custom Game",
            start=datetime(2024, 7, 1, tzinfo=timezone.utc),
            end=datetime(2024, 7, 31, tzinfo=timezone.utc),
            commit_points=commit_points,
            project_points=project_points,
        )
        assert game.commit_points == commit_points
        assert game.project_points == project_points

    async def test_invalid_dates_raises(self, game_service: GameService):
        with pytest.raises(ValueError, match="must be before end"):
            await game_service.create_game(
                name="Invalid Game",
                start=datetime(2024, 7, 31, tzinfo=timezone.utc),
                end=datetime(2024, 7, 1, tzinfo=timezone.utc),
            )


class TestAddCommit:

    async def test_creates_board_for_new_project(
        self,
        active_game: Game,
        game_service: GameService,
        make_commit,
        project,
        db_session,
    ):
        commit = await make_commit(hash="abc123")
        await game_service.add_commit(commit)

        stmt = select(Board).where(
            col(Board.game_id) == active_game.id,
            col(Board.project_id) == project.id,
        )
        result = await db_session.execute(stmt)
        board = result.scalar_one()

        assert board.commit_count == 1
        assert board.points == active_game.project_points + active_game.commit_points

    async def test_increments_existing_board(
        self,
        active_game: Game,
        game_service: GameService,
        make_commit,
        project,
        db_session,
    ):
        commit1 = await make_commit(hash="abc123")
        commit2 = await make_commit(hash="def456")

        await game_service.add_commit(commit1)
        await game_service.add_commit(commit2)

        stmt = select(Board).where(
            col(Board.game_id) == active_game.id,
            col(Board.project_id) == project.id,
        )
        result = await db_session.execute(stmt)
        board = result.scalar_one()

        assert board.commit_count == 2
        expected = active_game.project_points + (active_game.commit_points * 2)
        assert board.points == expected

    async def test_creates_player_for_new_user(
        self,
        active_game: Game,
        game_service: GameService,
        make_commit,
        user,
        db_session,
    ):
        commit = await make_commit(hash="abc123")
        await game_service.add_commit(commit)

        stmt = select(Player).where(
            col(Player.game_id) == active_game.id,
            col(Player.user_id) == user.id,
        )
        result = await db_session.execute(stmt)
        player = result.scalar_one()

        assert player.commit_count == 1
        assert player.project_count == 1
        assert player.points == active_game.project_points + active_game.commit_points

    async def test_updates_player_across_multiple_commits(
        self,
        active_game: Game,
        game_service: GameService,
        make_commit,
        user,
        db_session,
    ):
        for i in range(5):
            commit = await make_commit(hash=f"hash{i:04d}")
            await game_service.add_commit(commit)

        stmt = select(Player).where(
            col(Player.game_id) == active_game.id,
            col(Player.user_id) == user.id,
        )
        result = await db_session.execute(stmt)
        player = result.scalar_one()

        assert player.commit_count == 5
        assert player.project_count == 1
        expected = active_game.project_points + (active_game.commit_points * 5)
        assert player.points == expected

    async def test_tracks_multiple_projects(
        self,
        active_game: Game,
        game_service: GameService,
        make_commit,
        user,
        db_session,
    ):
        project2 = Project(
            name="second-project", url="https://github.com/test/second", slug="sluggy"
        )
        db_session.add(project2)
        await db_session.commit()

        commit1 = await make_commit(hash="abc123")
        commit2 = await make_commit(hash="def456", project_id=project2.id)

        await game_service.add_commit(commit1)
        await game_service.add_commit(commit2)

        stmt = select(Player).where(
            col(Player.game_id) == active_game.id,
            col(Player.user_id) == user.id,
        )
        result = await db_session.execute(stmt)
        player = result.scalar_one()

        assert player.project_count == 2
        expected = (active_game.project_points * 2) + (active_game.commit_points * 2)
        assert player.points == expected

    async def test_no_game_for_commit_timestamp(
        self, game_service: GameService, make_commit
    ):
        commit = await make_commit(
            hash="abc123",
            timestamp=datetime(2010, 1, 1, tzinfo=timezone.utc),
        )
        await game_service.add_commit(commit)
        assert commit.game_id is None

    async def test_no_user_for_commit(
        self, game_service: GameService, active_game: Game, make_commit
    ):
        commit = await make_commit(
            hash="abc123",
            email="foo@bar.com",
            user_id=None,
        )
        assert commit.user_id is None
        await game_service.add_commit(commit)
        assert commit.game_id == active_game.id


class TestLanguageBoards:

    @pytest.mark.parametrize(
        "languages",
        [
            pytest.param(["Python"], id="single"),
            pytest.param(["Python", "JavaScript"], id="multiple"),
            pytest.param(["Python", "JavaScript", "Rust"], id="three"),
        ],
    )
    async def test_creates_language_boards(
        self,
        active_game: Game,
        game_service: GameService,
        make_commit,
        db_session,
        languages: list[str],
    ):
        commit = await make_commit(hash="abc123", languages=languages)
        await game_service.add_commit(commit)

        stmt = select(LanguageBoard).where(col(LanguageBoard.game_id) == active_game.id)
        result = await db_session.execute(stmt)
        boards = result.scalars().all()

        assert len(boards) == len(languages)

    async def test_increments_language_board_counts(
        self,
        active_game: Game,
        game_service: GameService,
        make_commit,
        db_session,
    ):
        for i in range(3):
            commit = await make_commit(hash=f"hash{i:04d}", languages=["Python"])
            await game_service.add_commit(commit)

        stmt = (
            select(LanguageBoard)
            .join(Language)
            .where(
                col(LanguageBoard.game_id) == active_game.id,
                col(Language.name) == "Python",
            )
        )
        result = await db_session.execute(stmt)
        board = result.scalar_one()

        assert board.commit_count == 3
        assert board.points == active_game.commit_points * 3

    async def test_skips_empty_language_names(
        self,
        active_game: Game,
        game_service: GameService,
        make_commit,
        db_session,
    ):
        commit = await make_commit(hash="abc123", languages=["Python", "", None])
        await game_service.add_commit(commit)

        stmt = select(LanguageBoard).where(col(LanguageBoard.game_id) == active_game.id)
        result = await db_session.execute(stmt)
        boards = result.scalars().all()

        assert len(boards) == 1


class TestAIAnalysisAdjustment:

    @pytest.mark.parametrize(
        "initial_points,adjustment,expected_verified",
        [
            pytest.param(100, 0, 100, id="no-change"),
            pytest.param(100, -20, 80, id="penalty"),
            pytest.param(100, 10, 110, id="bonus"),
            pytest.param(50, -50, 0, id="full-penalty"),
            pytest.param(100, -150, -50, id="over-penalty"),
        ],
    )
    async def test_applies_adjustment(
        self,
        game_service: GameService,
        player_factory,
        initial_points: int,
        adjustment: int,
        expected_verified: int,
    ):
        player = await player_factory(potential_points=initial_points)
        await game_service.apply_ai_analysis_adjustment(str(player.id), adjustment)

        assert player.verified_points == expected_verified
        assert player.analysis_status == AnalysisStatus.COMPLETED
        assert player.last_analyzed_at is not None

    async def test_nonexistent_player_no_error(self, game_service: GameService):
        await game_service.apply_ai_analysis_adjustment(
            "00000000-0000-0000-0000-000000000000", points_adjustment=-10
        )


class TestLeaderboards:

    @pytest.mark.parametrize(
        "limit",
        [
            pytest.param(10, id="top-10"),
            pytest.param(25, id="top-25"),
            pytest.param(50, id="top-50"),
        ],
    )
    async def test_respects_limit(
        self,
        active_game: Game,
        game_service: GameService,
        limit: int,
    ):
        players = await game_service.get_leaderboard(str(active_game.id), limit=limit)
        assert len(players) <= limit

    async def test_orders_by_points_descending(
        self,
        active_game: Game,
        game_service: GameService,
        db_session,
    ):
        users = []
        for i, points in enumerate([50, 100, 25, 75]):
            user = await _create_user(db_session, f"user{i}")
            player = Player(
                game_id=active_game.id,
                user_id=user.id,
                potential_points=points,
                verified_points=points,
            )
            db_session.add(player)
            users.append((user, points))
        await db_session.commit()

        leaderboard = await game_service.get_leaderboard(str(active_game.id))

        points_order = [p.verified_points for p in leaderboard]
        assert points_order == sorted(points_order, reverse=True)

    async def test_project_leaderboard_ordering(
        self,
        active_game: Game,
        game_service: GameService,
        db_session,
    ):
        for i, points in enumerate([30, 60, 15, 45]):
            project = Project(
                name=f"project-{i}", url=f"https://example.com/{i}", slug=f"sluggy-{i}"
            )
            db_session.add(project)
            await db_session.flush()

            board = Board(
                game_id=active_game.id,
                project_id=project.id,
                potential_points=points,
                verified_points=points,
            )
            db_session.add(board)
        await db_session.commit()

        leaderboard = await game_service.get_project_leaderboard(str(active_game.id))

        points_order = [b.verified_points for b in leaderboard]
        assert points_order == sorted(points_order, reverse=True)

    async def test_language_leaderboard_ordering(
        self,
        active_game: Game,
        game_service: GameService,
        db_session,
    ):
        for name, points in [("Python", 100), ("Rust", 50), ("Go", 75)]:
            lang = Language(name=name)
            db_session.add(lang)
            await db_session.flush()

            board = LanguageBoard(
                game_id=active_game.id,
                language_id=lang.id,
                points=points,
            )
            db_session.add(board)
        await db_session.commit()

        leaderboard = await game_service.get_language_leaderboard(str(active_game.id))

        points_order = [lb.points for lb in leaderboard]
        assert points_order == sorted(points_order, reverse=True)

    async def test_excludes_inactive_users_from_leaderboard(
        self,
        active_game: Game,
        game_service: GameService,
        db_session,
    ):
        active_user = await _create_user(db_session, "active_user")
        inactive_user = await _create_user(db_session, "inactive_user")
        inactive_user.is_active = False

        for user, points in [(active_user, 100), (inactive_user, 200)]:
            player = Player(
                game_id=active_game.id,
                user_id=user.id,
                potential_points=points,
                verified_points=points,
            )
            db_session.add(player)
        await db_session.commit()

        leaderboard = await game_service.get_leaderboard(active_game.id)

        assert len(leaderboard) == 1
        assert leaderboard[0].user_id == active_user.id

    async def test_excludes_inactive_projects_from_leaderboard(
        self,
        active_game: Game,
        game_service: GameService,
        db_session,
    ):
        active_project = Project(
            name="active-project",
            url="https://github.com/test/active",
            slug="active-project",
            is_active=True,
        )
        inactive_project = Project(
            name="inactive-project",
            url="https://github.com/test/inactive",
            slug="inactive-project",
            is_active=False,
        )
        db_session.add(active_project)
        db_session.add(inactive_project)
        await db_session.flush()

        for project, points in [(active_project, 50), (inactive_project, 100)]:
            board = Board(
                game_id=active_game.id,
                project_id=project.id,
                potential_points=points,
                verified_points=points,
            )
            db_session.add(board)
        await db_session.commit()

        leaderboard = await game_service.get_project_leaderboard(active_game.id)

        assert len(leaderboard) == 1
        assert leaderboard[0].project_id == active_project.id

    async def test_deactivated_user_removed_from_leaderboard(
        self,
        active_game: Game,
        game_service: GameService,
        db_session,
    ):
        user = await _create_user(db_session, "testuser")
        player = Player(
            game_id=active_game.id,
            user_id=user.id,
            potential_points=100,
            verified_points=100,
        )
        db_session.add(player)
        await db_session.commit()

        # Verify user is on leaderboard
        leaderboard = await game_service.get_leaderboard(active_game.id)
        assert len(leaderboard) == 1

        # Deactivate user
        await game_service.deactivate_user(user.id)
        await db_session.commit()

        # Verify user is no longer on leaderboard
        leaderboard = await game_service.get_leaderboard(active_game.id)
        assert len(leaderboard) == 0

    async def test_deactivated_project_removed_from_leaderboard(
        self,
        active_game: Game,
        game_service: GameService,
        db_session,
    ):
        project = Project(
            name="test-project",
            url="https://github.com/test/project",
            slug="test-project",
        )
        db_session.add(project)
        await db_session.flush()

        board = Board(
            game_id=active_game.id,
            project_id=project.id,
            potential_points=100,
            verified_points=100,
        )
        db_session.add(board)
        await db_session.commit()

        # Verify project is on leaderboard
        leaderboard = await game_service.get_project_leaderboard(active_game.id)
        assert len(leaderboard) == 1

        # Deactivate project
        await game_service.deactivate_project(project.id)
        await db_session.commit()

        # Verify project is no longer on leaderboard
        leaderboard = await game_service.get_project_leaderboard(active_game.id)
        assert len(leaderboard) == 0


class TestDeactivateUser:

    async def test_deactivates_existing_user(
        self,
        game_service: GameService,
        db_session,
    ):
        user = await _create_user(db_session, "testuser")
        await db_session.commit()

        result = await game_service.deactivate_user(user.id)

        assert result is True
        assert user.is_active is False

    async def test_deactivates_with_reason(
        self,
        game_service: GameService,
        db_session,
    ):
        user = await _create_user(db_session, "testuser")
        await db_session.commit()

        result = await game_service.deactivate_user(user.id, reason="Spam commits")

        assert result is True
        assert user.is_active is False
        assert user.banned_reason == "Spam commits"
        assert user.banned_at is not None

    async def test_nonexistent_user_returns_false(
        self,
        game_service: GameService,
    ):
        fake_id = uuid.uuid4()
        result = await game_service.deactivate_user(fake_id)

        assert result is False

    async def test_already_inactive_user(
        self,
        game_service: GameService,
        db_session,
    ):
        user = await _create_user(db_session, "testuser")
        user.is_active = False
        await db_session.commit()

        result = await game_service.deactivate_user(user.id)

        assert result is True
        assert user.is_active is False


class TestDeactivateProject:

    async def test_deactivates_existing_project(
        self,
        game_service: GameService,
        db_session,
    ):
        project = Project(
            name="test-project",
            url="https://github.com/test/project",
            slug="test-project",
        )
        db_session.add(project)
        await db_session.commit()

        result = await game_service.deactivate_project(project.id)

        assert result is True
        assert project.is_active is False

    async def test_deactivates_with_reason_logged(
        self,
        game_service: GameService,
        db_session,
    ):
        project = Project(
            name="test-project",
            url="https://github.com/test/project",
            slug="test-project",
        )
        db_session.add(project)
        await db_session.commit()

        result = await game_service.deactivate_project(
            project.id, reason="Gaming detected"
        )

        assert result is True
        assert project.is_active is False

    async def test_nonexistent_project_returns_false(
        self,
        game_service: GameService,
    ):
        fake_id = uuid.uuid4()
        result = await game_service.deactivate_project(fake_id)

        assert result is False


class TestClaimOrphanedCommits:

    async def test_does_nothing_without_active_game(
        self,
        make_commit,
        user: User,
        user_service: UserService,
        game_service: GameService,
    ):
        commit1 = await make_commit(hash="abc123", user_id=None)
        await game_service.add_commit(commit1)
        await user_service.upsert_identifier(
            user, IdentifierType.EMAIL, commit1.email, data={}
        )
        added = await claim_orphan_commits_task(user.id, emails=[commit1.email])
        assert added == 0

    async def test_does_nothing_without_emails(self, user: User):
        added = await claim_orphan_commits_task(user.id, emails=[])
        assert added == 0

    async def test_does_nothing_if_no_orphans_found(
        self,
        user: User,
        make_commit,
        game_service: GameService,
        db_session,
    ):
        now = times.now()
        await game_service.create_julython_game(
            year=now.year,
            month=now.month,
            is_active=True,
        )
        commit1 = await make_commit(hash="abc123", user_id=None, timestamp=now)
        await game_service.add_commit(commit1)

        added = await claim_orphan_commits_task(user.id, emails=["jane@doe.com"])
        assert added == 0

    async def test_adds_player_to_game(
        self,
        make_commit,
        user: User,
        user_service: UserService,
        game_service: GameService,
        db_session,
    ):
        # Create an active game for the current time
        now = times.now()
        active_game = await game_service.create_julython_game(
            year=now.year,
            month=now.month,
            is_active=True,
        )
        commit1 = await make_commit(hash="abc123", user_id=None, timestamp=now)
        commit2 = await make_commit(hash="def456", user_id=None, timestamp=now)

        await game_service.add_commit(commit1)
        await game_service.add_commit(commit2)

        stmt = select(Player).where(
            col(Player.game_id) == active_game.id,
            col(Player.user_id) == user.id,
        )
        result = await db_session.execute(stmt)
        player = result.scalar_one_or_none()
        assert player is None

        await user_service.upsert_identifier(
            user, IdentifierType.EMAIL, commit1.email, data={}
        )
        await db_session.commit()

        added = await claim_orphan_commits_task(user.id, emails=[commit1.email])
        assert added == 2

        result = await db_session.execute(stmt)
        player = result.scalar_one_or_none()
        assert player.user_id == user.id
