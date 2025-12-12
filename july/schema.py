from datetime import datetime
from typing import Optional

from pydantic import BaseModel
from july.db.models import ReportType, ReportStatus


class AnalysisSubmission(BaseModel):
    repo_full_name: str
    repo_url: str
    team_id: Optional[int] = None
    commit_count: int
    date_range_start: datetime
    date_range_end: datetime
    ai_model: str = "rule_based"
    quality_score: float
    quality_reasoning: Optional[str] = None
    dev_style: str
    style_reasoning: Optional[str] = None
    authenticity_score: float = 100.0
    authenticity_reasoning: Optional[str] = None
    ai_insights: Optional[str] = None
    red_flags: Optional[list] = None
    key_strengths: Optional[list] = None
    recommendations: Optional[list] = None
    streak_days: Optional[int] = None
    avg_commits_per_day: Optional[float] = None
    commit_patterns: Optional[dict] = None
    language_breakdown: Optional[dict] = None
    peak_activity_hours: Optional[dict] = None
    consistency_score: Optional[float] = None
    ghost_commit_ratio: Optional[float] = None
    is_public: bool = True


class TeamCreate(BaseModel):
    name: str
    description: Optional[str] = None
    avatar_url: Optional[str] = None
    is_public: bool = True


class ReportCreate(BaseModel):
    analysis_id: Optional[int] = None
    reported_user_id: Optional[int] = None
    report_type: ReportType
    reason: str


class ReportAction(BaseModel):
    status: ReportStatus
    moderator_notes: Optional[str] = None
    ban_user: bool = False
    remove_analysis: bool = False
    ban_reason: Optional[str] = None
