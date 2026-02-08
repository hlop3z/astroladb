use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use uuid::Uuid;

use crate::models::{User, UserBase};

type AppState = ();

// user handlers

pub async fn list_user(
    State(_state): State<AppState>,
) -> impl IntoResponse {
    (StatusCode::NOT_IMPLEMENTED, Json("not implemented"))
}


pub async fn get_user(
    State(_state): State<AppState>,
    Path(id): Path<Uuid>,
) -> impl IntoResponse {
    (StatusCode::NOT_IMPLEMENTED, Json("not implemented"))
}


pub async fn create_user(
    State(_state): State<AppState>,
    Json(_body): Json<UserBase>,
) -> impl IntoResponse {
    (StatusCode::NOT_IMPLEMENTED, Json("not implemented"))
}


pub async fn update_user(
    State(_state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(_body): Json<UserBase>,
) -> impl IntoResponse {
    (StatusCode::NOT_IMPLEMENTED, Json("not implemented"))
}


pub async fn delete_user(
    State(_state): State<AppState>,
    Path(id): Path<Uuid>,
) -> impl IntoResponse {
    StatusCode::NOT_IMPLEMENTED
}
