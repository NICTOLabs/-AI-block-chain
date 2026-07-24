use reqwest::blocking::Client;
use serde_json::Value;

/// A minimal Rust client for the Tender node RPC layer.
#[derive(Debug, Clone)]
pub struct TenderClient {
    base_url: String,
    client: Client,
}

impl TenderClient {
    /// Create a new client targeting the supplied base URL.
    pub fn new(base_url: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into(),
            client: Client::new(),
        }
    }

    /// Return the health payload from the node.
    pub fn health(&self) -> Result<Value, String> {
        let url = format!("{}/health", self.base_url);
        let response = self.client.get(&url).send().map_err(|err| format!("request failed: {err}"))?;

        if !response.status().is_success() {
            return Err(format!("health request failed: {}", response.status()));
        }

        let payload = response.json::<Value>().map_err(|err| format!("invalid health response: {err}"))?;
        Ok(payload)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::{Read, Write};
    use std::net::TcpListener;
    use std::thread;

    #[test]
    fn health_returns_json_payload() {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind test listener");
        let addr = listener.local_addr().expect("read test listener addr");

        let server = thread::spawn(move || {
            let (mut stream, _) = listener.accept().expect("accept test connection");
            let mut buffer = [0u8; 1024];
            let _ = stream.read(&mut buffer).expect("read request");

            let response = b"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 11\r\n\r\n{\"ok\":true}";
            stream.write_all(response).expect("write response");
        });

        let client = TenderClient::new(format!("http://{addr}"));
        let payload = client.health().expect("health request should succeed");

        assert_eq!(payload["ok"], true);
        server.join().expect("server thread should finish");
    }
}
