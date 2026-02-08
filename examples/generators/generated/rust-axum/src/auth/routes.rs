use axum::{routing::{delete, get, patch, post}, Router};

use crate::auth::handlers::*;

pub fn auth_routes() -> Router {
    Router::new()
        .route("/user", get(list_user).post(create_user))
        .route("/user/:id", get(get_user).patch(update_user).delete(delete_user))
}
