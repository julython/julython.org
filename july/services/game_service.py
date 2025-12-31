from datetime import datetime, timezone
from typing import Optional
from uuid import UUID

from sqlalchemy import select, func, update
from sqlalchemy.orm import selectinload
from sqlalchemy.ext.asyncio import AsyncSession
from sqlmodel import col
from structlog.stdlib import get_logger

from july.db.models import (
    Game,
    Player,
    Board,
    Commit,
    Language,
    LanguageBoard,
    AnalysisStatus,
    Project,
    User,
)
from july.globals import context
from july.types import Identifier
from july.utils import times

logger = get_logger(__name__)


class GameService:
    """Service for handling game scoring logic"""

    def __init__(self, session: AsyncSession):
        self.session = session

    async def create_game(
        self,
        name: str,
        start: datetime,
        end: datetime,
        commit_points: int = 1,
        project_points: int = 10,
        is_active: bool = False,
        deactivate_others: bool = False,
    ) -> Game:
        """
        Create and return a new game.

        Args:
            name: Display name for the game (e.g., "Julython 2024")
            start: Game start datetime
            end: Game end datetime
            commit_points: Points awarded per commit (default: 1)
            project_points: Points awarded per unique project (default: 10)
            is_active: Whether this game is currently active (default: False)
            deactivate_others: If True and is_active=True, deactivate all other games

        Returns:
            The created Game instance

        Raises:
            ValueError: If start >= end

        Example:
            game = await game_service.create_game(
                name="Julython 2024",
                start=datetime(2024, 7, 1, tzinfo=timezone.utc),
                end=datetime(2024, 7, 31, 23, 59, 59, tzinfo=timezone.utc),
                is_active=True,
                deactivate_others=True,
            )
        """
        # Validate date range
        if start >= end:
            raise ValueError(f"Game start ({start}) must be before end ({end})")

        # Deactivate other games if requested
        if is_active and deactivate_others:
            statement = (
                select(Game).where(col(Game.is_active) == True).with_for_update()
            )
            result = await self.session.execute(statement)
            active_games = result.scalars().all()

            for game in active_games:
                game.is_active = False
                logger.info(f"Deactivated game: {game.name}")

        # Create the new game
        game = Game(
            name=name,
            start=start,
            end=end,
            commit_points=commit_points,
            project_points=project_points,
            is_active=is_active,
        )

        self.session.add(game)

        logger.info(f"Created game: {game.name} ({game.start} - {game.end})")

        return game

    async def create_julython_game(
        self,
        year: int,
        month: int = 7,  # July by default
        is_active: bool = False,
        deactivate_others: bool = False,
    ) -> Game:
        """
        Create a Julython or J(an)ulython game with standard settings.

        Args:
            year: The year of the event
            month: Month number (7 for July, 1 for January)
            is_active: Whether this game is currently active
            deactivate_others: If True and is_active=True, deactivate all other games

        Returns:
            The created Game instance

        Example:
            # Create Julython 2024 (July)
            game = await game_service.create_julython_game(2024)

            # Create J(an)ulython 2024 (January)
            game = await game_service.create_julython_game(2024, month=1)
        """
        if month == 7:
            name = f"Julython {year}"
            start = datetime(year, 7, 1, tzinfo=timezone.utc)
            end = datetime(year, 7, 31, 23, 59, 59, tzinfo=timezone.utc)
        elif month == 1:
            name = f"J(an)ulython {year}"
            start = datetime(year, 1, 1, tzinfo=timezone.utc)
            end = datetime(year, 1, 31, 23, 59, 59, tzinfo=timezone.utc)
        elif 8 <= month <= 12:
            name = f"Testathon Fall {year}"
            start = datetime(year, 8, 1, tzinfo=timezone.utc)
            end = datetime(year, 12, 31, 23, 59, 59, tzinfo=timezone.utc)
        elif 2 <= month <= 6:
            name = f"Testathon Spring {year}"
            start = datetime(year, 2, 1, tzinfo=timezone.utc)
            end = datetime(year, 6, 30, 23, 59, 59, tzinfo=timezone.utc)
        else:
            raise ValueError(f"Invalid month {month}. Use 7 for July or 1 for January")

        return await self.create_game(
            name=name,
            start=start,
            end=end,
            is_active=is_active,
            deactivate_others=deactivate_others,
        )

    async def get_active_game(self, now: Optional[datetime] = None) -> Optional[Game]:
        """Returns the active game or None."""
        now = now or times.now()

        statement = (
            select(Game)
            .where(
                col(Game.start) <= now,
                col(Game.end) >= now,
                col(Game.is_active) == True,
            )
            .order_by(col(Game.start).desc())
            .limit(1)
        )
        result = await self.session.execute(statement)
        return result.scalar_one_or_none()

    async def get_active_or_latest_game(
        self, now: Optional[datetime] = None
    ) -> Optional[Game]:
        """Return an active game or the latest one."""
        now = now or times.now()

        # Try to get active game first
        game = await self.get_active_game(now)

        if game is None:
            # Get the most recent completed game
            statement = (
                select(Game)
                .where(col(Game.end) <= now)
                .order_by(col(Game.end).desc())
                .limit(1)
            )
            result = await self.session.execute(statement)
            game = result.scalar_one_or_none()

        return game

    async def add_commit(self, commit: Commit) -> None:
        """
        Add a commit to the game and update scores.

        Args:
            commit: The commit to add
        """
        # Get the active game at the time of commit
        game = await self.get_active_game(now=commit.timestamp)

        if game is None:
            logger.info(
                f"No active game for commit {commit.hash} at {commit.timestamp}"
            )
            return

        # Update the commit's game_id
        commit.game_id = game.id

        # Update boards and player scores
        await self._add_points_to_board(game, commit)
        await self._add_points_to_language_boards(game, commit)

        if commit.user_id:
            await self._add_points_to_player(game, commit.user_id)

    async def claim_orphan_commits(self, user_id: Identifier, emails: list[str]) -> int:
        """
        Find commits matching the user's emails and assign them.
        Returns the number of commits claimed.
        """
        if not emails:
            return 0

        # Only update the latest game
        game = await self.get_active_or_latest_game()
        if game is None:
            return 0

        user_uuid = UUID(user_id) if isinstance(user_id, str) else user_id

        # Find orphan commits matching any of the user's emails
        stmt = (
            select(Commit)
            .where(
                col(Commit.user_id).is_(None),
                col(Commit.email).in_(emails),
                col(Commit.game_id) == game.id,
            )
            .with_for_update(skip_locked=True)
        )
        result = await self.session.execute(stmt)
        orphan_commits = list(result.scalars().all())

        if not orphan_commits:
            return 0

        # Bulk update user_id on all orphan commits
        commit_ids = [c.id for c in orphan_commits]
        await self.session.execute(
            update(Commit)
            .where(col(Commit.id).in_(commit_ids))
            .values(user_id=user_uuid)
        )

        # Single recalculate covers everything
        player = await self._get_or_create_player(game, user_uuid)
        await self._recalculate_player_points(game, player)

        return len(orphan_commits)

    async def _get_or_create_player(self, game: Game, user_id: UUID) -> Player:
        stmt = (
            select(Player)
            .where(col(Player.game_id) == game.id, col(Player.user_id) == user_id)
            .with_for_update()
        )
        result = await self.session.execute(stmt)
        player = result.scalar_one_or_none()

        if player is None:
            player = Player(
                game_id=game.id,
                user_id=user_id,
                points=0,
                potential_points=0,
                commit_count=0,
                project_count=0,
            )
            self.session.add(player)
            await self.session.flush()

        return player

    async def _add_points_to_board(self, game: Game, commit: Commit) -> Board:
        """Create or update a board for a project."""
        # Lock the row for update to prevent race conditions
        statement = (
            select(Board)
            .where(
                col(Board.game_id) == game.id,
                col(Board.project_id) == commit.project_id,
            )
            .with_for_update()
        )
        result = await self.session.execute(statement)
        board = result.scalar_one_or_none()

        if board is None:
            # Create new board with initial points
            board = Board(
                game_id=game.id,
                project_id=commit.project_id,
                points=game.project_points + game.commit_points,
                potential_points=game.project_points + game.commit_points,
                commit_count=1,
                contributor_count=1,
            )
            self.session.add(board)
        else:
            # Update existing board
            board.points += game.commit_points
            board.potential_points += game.commit_points
            board.commit_count += 1

        return board

    async def _add_points_to_language_boards(self, game: Game, commit: Commit) -> None:
        """Create or update language boards for commit languages."""
        if not commit.languages:
            return

        for language_name in commit.languages:
            if not language_name:
                continue

            # Get or create language
            lang_statement = select(Language).where(col(Language.name) == language_name)
            lang_result = await self.session.execute(lang_statement)
            language = lang_result.scalar_one_or_none()

            if language is None:
                language = Language(name=language_name)
                self.session.add(language)
                await self.session.flush()  # Get the ID

            # Lock and get/create language board
            board_statement = (
                select(LanguageBoard)
                .where(
                    col(LanguageBoard.game_id) == game.id,
                    col(LanguageBoard.language_id) == language.id,
                )
                .with_for_update()
            )
            board_result = await self.session.execute(board_statement)
            language_board = board_result.scalar_one_or_none()

            if language_board is None:
                language_board = LanguageBoard(
                    game_id=game.id,
                    language_id=language.id,
                    points=game.commit_points,
                    commit_count=1,
                )
                self.session.add(language_board)
            else:
                language_board.points += game.commit_points
                language_board.commit_count += 1

    async def _add_points_to_player(self, game: Game, user_id: UUID) -> None:
        """Create or update a player's score."""

        stmt = (
            select(Player)
            .where(col(Player.game_id) == game.id, col(Player.user_id) == user_id)
            .with_for_update()
        )
        result = await self.session.execute(stmt)
        player = result.scalar_one_or_none()

        if player is None:
            player = Player(
                game_id=game.id,
                user_id=user_id,
                points=0,
                potential_points=0,
                commit_count=0,
                project_count=0,
            )
            self.session.add(player)

        await self._recalculate_player_points(game, player)

    async def _recalculate_player_points(self, game: Game, player: Player) -> None:
        """Recalculate a player's total points from commits."""
        stmt = select(
            func.count(col(Commit.id)).label("commit_count"),
            func.count(func.distinct(col(Commit.project_id))).label("project_count"),
        ).where(
            col(Commit.user_id) == player.user_id,
            col(Commit.game_id) == game.id,
        )
        result = await self.session.execute(stmt)
        row = result.one()

        player.commit_count = row.commit_count
        player.project_count = row.project_count
        player.points = (
            row.commit_count * game.commit_points
            + row.project_count * game.project_points
        )
        player.potential_points = player.points

    async def apply_ai_analysis_adjustment(
        self, player_id: Identifier, points_adjustment: int
    ) -> None:
        """
        Apply AI analysis score adjustment to a player.

        Args:
            player_id: The player's UUID
            points_adjustment: Points to add/subtract (can be negative)
        """
        statement = select(Player).where(col(Player.id) == player_id).with_for_update()
        result = await self.session.execute(statement)
        player = result.scalar_one_or_none()

        if player is None:
            logger.warning(f"Player {player_id} not found for adjustment")
            return

        player.verified_points = player.potential_points + points_adjustment
        player.analysis_status = AnalysisStatus.COMPLETED
        player.last_analyzed_at = times.now()

    async def get_leaderboard(
        self, game_id: Identifier, limit: int = 50
    ) -> list[Player]:
        """Get the top players for a game."""
        statement = (
            select(Player)
            .join(User, col(Player.user_id) == col(User.id))
            .where(
                col(Player.game_id) == game_id,
                col(User.is_active) == True,
            )
            .options(selectinload(Player.user))  # type: ignore
            .order_by(
                col(Player.verified_points).desc(), col(Player.potential_points).desc()
            )
            .limit(limit)
        )
        result = await self.session.execute(statement)
        return list(result.scalars().all())

    async def get_project_leaderboard(
        self, game_id: Identifier, limit: int = 50
    ) -> list[Board]:
        """Get the top projects for a game."""
        statement = (
            select(Board)
            .join(Project, col(Board.project_id) == col(Project.id))
            .where(
                col(Board.game_id) == game_id,
                col(Project.is_active) == True,
            )
            .options(selectinload(Board.project))  # type: ignore
            .order_by(
                col(Board.verified_points).desc(), col(Board.potential_points).desc()
            )
            .limit(limit)
        )
        result = await self.session.execute(statement)
        return list(result.scalars().all())

    async def get_language_leaderboard(
        self, game_id: Identifier, limit: int = 50
    ) -> list[LanguageBoard]:
        """Get the top languages for a game."""
        statement = (
            select(LanguageBoard)
            .where(col(LanguageBoard.game_id) == game_id)
            .order_by(col(LanguageBoard.points).desc())
            .limit(limit)
        )
        result = await self.session.execute(statement)
        return list(result.scalars().all())

    async def deactivate_user(
        self, user_id: Identifier, reason: Optional[str] = None
    ) -> bool:
        """
        Mark a user as inactive, removing them from leaderboards.

        Args:
            user_id: The user's UUID
            reason: Optional reason for deactivation

        Returns:
            True if user was found and deactivated, False otherwise
        """
        statement = select(User).where(col(User.id) == user_id).with_for_update()
        result = await self.session.execute(statement)
        user = result.scalar_one_or_none()

        if user is None:
            logger.warning(f"User {user_id} not found for deactivation")
            return False

        user.is_active = False
        if reason:
            user.banned_reason = reason
            user.banned_at = times.now()

        logger.info(
            f"Deactivated user: {user.name} ({user_id}) - {reason or 'no reason'}"
        )
        return True

    async def deactivate_project(
        self, project_id: Identifier, reason: Optional[str] = None
    ) -> bool:
        """
        Mark a project as inactive, removing it from leaderboards.

        Args:
            project_id: The project's UUID
            reason: Optional reason for deactivation (logged only)

        Returns:
            True if project was found and deactivated, False otherwise
        """
        statement = (
            select(Project).where(col(Project.id) == project_id).with_for_update()
        )
        result = await self.session.execute(statement)
        project = result.scalar_one_or_none()

        if project is None:
            logger.warning(f"Project {project_id} not found for deactivation")
            return False

        project.is_active = False

        logger.info(
            f"Deactivated project: {project.name} ({project_id}) - {reason or 'no reason'}"
        )
        return True


async def claim_orphan_commits_task(
    user_id: Identifier,
    emails: list[str],
) -> None:
    """
    Background task to claim orphan commits for a newly registered user.

    Uses its own session since background tasks outlive the request.
    """
    async with context.db_session() as session:
        try:
            game_service = GameService(session)
            claimed = await game_service.claim_orphan_commits(user_id, emails)

            if claimed > 0:
                await session.commit()
                logger.info(
                    f"Claimed orphan {claimed} commits for {user_id}",
                    user_id=str(user_id),
                    commit_count=claimed,
                )
            else:
                logger.debug("No orphan commits found", user_id=str(user_id))

        except Exception:
            logger.exception("Failed to claim orphan commits", user_id=str(user_id))
            await session.rollback()
