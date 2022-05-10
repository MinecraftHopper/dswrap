#![warn(clippy::all, clippy::pedantic, clippy::nursery)]

use std::net::SocketAddr;

use axum::{
    body::StreamBody,
    extract::Path,
    http::{HeaderMap, HeaderValue, StatusCode},
    response::{IntoResponse, Redirect},
    routing::get,
};

#[tokio::main]
async fn main() {
    dotenv::dotenv().ok();
    let port: u16 = std::env::var("PORT")
        .expect("PORT must be set to run the pastebin")
        .parse()
        .expect("PORT invalid");
    let http_client = reqwest::Client::builder()
        .referer(false)
        .user_agent(concat!(
            env!("CARGO_PKG_NAME"),
            "/",
            env!("CARGO_PKG_VERSION")
        ))
        .build()
        .expect("Failed to create HTTP client");
    let app = axum::Router::new()
        .route(
            "/",
            get(|| async { Redirect::permanent("https://github.com/minecrafthopper/dswrap") }),
        )
        .route(
            "/:channelid/:messageid/:filename",
            get(move |path| get_file(path, http_client)),
        );
    let listen = SocketAddr::from(([0, 0, 0, 0], port));
    println!("[INFO] Listening on http://{}", &listen);
    axum::Server::bind(&listen)
        .serve(app.into_make_service())
        .await
        .expect("Failed to start the server");
}

async fn get_file(
    Path((channelid, messageid, filename)): Path<(String, String, String)>,
    http: reqwest::Client,
) -> Result<impl IntoResponse, Error> {
    let req = http
        .get(format!(
            "https://cdn.discordapp.net/attachments/{}/{}/{}",
            channelid, messageid, filename
        ))
        .build()?;
    let resp = http.execute(req).await?;
    let headers = resp.headers();
    let resp_headers = {
        let mut head = HeaderMap::new();
        let backup_value = HeaderValue::from_static("application/octet-stream");
        head.insert(
            "Content-Type",
            headers.get("Content-Type").unwrap_or(&backup_value).clone(),
        );
        head
    };
    let data = resp.bytes_stream();
    Ok((resp_headers, StreamBody::new(data)))
}

enum Error {
    NotFound,
    Reqwest(reqwest::Error),
}

impl From<reqwest::Error> for Error {
    fn from(e: reqwest::Error) -> Self {
        Self::Reqwest(e)
    }
}

impl From<std::string::FromUtf8Error> for Error {
    fn from(_: std::string::FromUtf8Error) -> Self {
        Self::NotFound
    }
}

impl axum::response::IntoResponse for Error {
    fn into_response(self) -> axum::response::Response {
        let (error, status): (String, StatusCode) = match self {
            Error::Reqwest(e) => (
                format!("Discord returned an error: {:?}", e),
                StatusCode::INTERNAL_SERVER_ERROR,
            ),

            Error::NotFound => ("404 paste not found".to_string(), StatusCode::NOT_FOUND),
        };
        axum::response::Response::builder()
            .status(status)
            .body(axum::body::boxed(axum::body::Full::from(error)))
            .unwrap()
    }
}
