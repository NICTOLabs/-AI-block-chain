use serde_json::Value;
use std::process::Command;

/// A minimal Rust client for the Tender node RPC layer.
#[derive(Debug, Clone)]
pub struct TenderClient {
    base_url: String,
}

impl TenderClient {
    /// Create a new client targeting the supplied base URL.
    pub fn new(base_url: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into(),
        }
    }

    /// Return the health payload from the node.
    pub fn health(&self) -> Result<Value, String> {
        let output = Command::new("curl")
            .arg("-s")
            .arg("-H")
            .arg("Accept: application/json")
            .arg(format!("{}/health", self.base_url))
            .output()
            .map_err(|err| format!("failed to invoke curl: {err}"))?;

        if !output.status.success() {
            return Err(format!(
                "health request failed: {}",
                String::from_utf8_lossy(&output.stderr)
            ));
        }

        let body = String::from_utf8(output.stdout)
            .map_err(|err| format!("health response was not valid UTF-8: {err}"))?;

        serde_json::from_str(&body).map_err(|err| format!("invalid health response: {err}"))
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
