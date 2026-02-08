from __future__ import annotations

from datetime import date, time, datetime
from decimal import Decimal
from enum import Enum
from typing import Optional
from uuid import UUID

from pydantic import BaseModel

class RoleEnum(str, Enum):
    ADMIN = "admin"
    EDITOR = "editor"
    VIEWER = "viewer"

class UserBase(BaseModel):
    age: Optional[int] = None
    avatar: Optional[str] = None
    balance: Decimal = Decimal("0")
    bio: Optional[str] = None
    birthdate: Optional[date] = None
    external_id: Optional[UUID] = None
    is_active: bool = True
    last_seen: Optional[datetime] = None
    login_time: Optional[time] = None
    role: RoleEnum = RoleEnum.VIEWER
    score: Optional[float] = None
    settings: Optional[dict] = None
    username: str


class User(UserBase):
    id: UUID
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True
