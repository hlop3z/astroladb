# Auto-generated Python types from database schema
# Do not edit manually

from __future__ import annotations
from dataclasses import dataclass
from typing import Optional, Any
from datetime import datetime, date, time
from enum import Enum


class AuthUserRole(str, Enum):
    ADMIN = "admin"
    EDITOR = "editor"
    VIEWER = "viewer"


@dataclass
class AuthUser:
    balance: str
    id: str
    is_active: bool
    role: AuthUserRole
    username: str
    created_at: datetime
    updated_at: datetime
    age: Optional[int] = None
    avatar: Optional[bytes] = None
    bio: Optional[str] = None
    birthdate: Optional[date] = None
    external_id: Optional[str] = None
    last_seen: Optional[datetime] = None
    login_time: Optional[time] = None
    score: Optional[float] = None
    settings: Optional[Any] = None


# Schema URI to type name mapping
TYPES: dict[str, str] = {
    "auth.user": "AuthUser",
}
