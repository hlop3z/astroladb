// Example code that deserializes and serializes the model.
// extern crate serde;
// #[macro_use]
// extern crate serde_derive;
// extern crate serde_json;
//
// use generated_module::types;
//
// fn main() {
//     let json = r#"{"answer": 42}"#;
//     let model: types = serde_json::from_str(&json).unwrap();
// }

use miniserde::{json, Value};
use std::collections::HashMap;

#[derive(Type)]
pub struct Types {
    blog_comment: Option<BlogComment>,

    blog_post: Option<BlogPost>,

    blog_user: Option<BlogUser>,
}

#[derive(Type)]
pub struct BlogComment {
    author_id: String,

    body: String,

    created_at: String,

    id: String,

    post_id: String,

    updated_at: String,
}

#[derive(Type)]
pub struct BlogPost {
    author_id: String,

    content: String,

    created_at: String,

    featured: bool,

    id: String,

    slug: String,

    status: String,

    title: String,

    updated_at: String,
}

#[derive(Type)]
pub struct BlogUser {
    balance: f64,

    bio: Option<String>,

    created_at: String,

    display_name: String,

    email: String,

    id: String,

    level: i64,

    money: String,

    the_date: String,

    the_day: String,

    the_file: String,

    the_object: HashMap<String, Option<Value>>,

    the_time: String,

    updated_at: String,

    username: String,
}
