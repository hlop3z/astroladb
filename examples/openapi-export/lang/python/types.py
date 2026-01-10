from dataclasses import dataclass
from uuid import UUID
from datetime import datetime
from typing import Dict, Any, Optional


@dataclass
class BlogComment:
    author_id: UUID
    body: str
    created_at: datetime
    id: UUID
    post_id: UUID
    updated_at: datetime


@dataclass
class BlogPost:
    author_id: UUID
    content: str
    created_at: datetime
    featured: bool
    id: UUID
    slug: str
    status: str
    title: str
    updated_at: datetime


@dataclass
class BlogUser:
    balance: float
    created_at: datetime
    display_name: str
    email: str
    id: UUID
    level: int
    money: str
    the_date: datetime
    the_day: datetime
    the_file: str
    the_object: Dict[str, Any]
    the_time: datetime
    updated_at: datetime
    username: str
    bio: Optional[str] = None


@dataclass
class Types:
    blog_comment: Optional[BlogComment] = None
    blog_post: Optional[BlogPost] = None
    blog_user: Optional[BlogUser] = None
