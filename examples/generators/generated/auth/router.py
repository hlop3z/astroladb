from __future__ import annotations

from uuid import UUID

from fastapi import APIRouter, HTTPException

from .models import (
    User,
    UserBase,
)


router = APIRouter(prefix="/auth", tags=["auth"])


@router.get("/user", response_model=list[User])
async def list_user():
    # TODO: implement
    raise HTTPException(501, "not implemented")


@router.get("/user/{item_id}", response_model=User)
async def get_user(item_id: UUID):
    # TODO: implement
    raise HTTPException(501, "not implemented")


@router.post("/user", response_model=User, status_code=201)
async def create_user(body: UserBase):
    # TODO: implement
    raise HTTPException(501, "not implemented")


@router.patch("/user/{item_id}", response_model=User)
async def update_user(item_id: UUID, body: UserBase):
    # TODO: implement
    raise HTTPException(501, "not implemented")


@router.delete("/user/{item_id}", status_code=204)
async def delete_user(item_id: UUID):
    # TODO: implement
    raise HTTPException(501, "not implemented")


