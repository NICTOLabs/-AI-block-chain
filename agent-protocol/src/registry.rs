use serde::{Deserialize, Serialize};
use thiserror::Error;

/// A 32-byte address used for registry participants.
type Address = [u8; 32];

/// An on-chain model record.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ModelRecord {
    pub model_id: String,
    pub provider: Address,
    pub metadata_uri: String,
    pub version: u32,
    pub price_per_call: u128,
    pub is_active: bool,
    pub registered_at: u64,
}

/// A basic in-memory registry for AI model metadata.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ModelRegistry {
    models: Vec<ModelRecord>,
}

/// Errors emitted by the registry module.
#[derive(Debug, Error)]
pub enum RegistryError {
    #[error("model already exists")]
    AlreadyExists,
    #[error("model not found")]
    NotFound,
}

impl ModelRegistry {
    /// Register a new model.
    pub fn register_model(&mut self, record: ModelRecord) -> Result<(), RegistryError> {
        if self.models.iter().any(|m| m.model_id == record.model_id) {
            return Err(RegistryError::AlreadyExists);
        }
        self.models.push(record);
        Ok(())
    }

    /// Mark a registered model as inactive.
    pub fn deprecate_model(&mut self, model_id: &str) -> Result<(), RegistryError> {
        let Some(model) = self.models.iter_mut().find(|m| m.model_id == model_id) else {
            return Err(RegistryError::NotFound);
        };
        model.is_active = false;
        Ok(())
    }

    /// Query a model by id.
    pub fn query_model(&self, model_id: &str) -> Result<ModelRecord, RegistryError> {
        self.models
            .iter()
            .find(|m| m.model_id == model_id)
            .cloned()
            .ok_or(RegistryError::NotFound)
    }

    /// Return all models matching a simple filter.
    pub fn list_models(&self, active_only: bool) -> Vec<ModelRecord> {
        self.models
            .iter()
            .filter(|m| !active_only || m.is_active)
            .cloned()
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn registers_and_queries_models() {
        let mut registry = ModelRegistry::default();
        let record = ModelRecord {
            model_id: "gpt-4o-mini".into(),
            provider: [1u8; 32],
            metadata_uri: "ipfs://abc".into(),
            version: 1,
            price_per_call: 42,
            is_active: true,
            registered_at: 7,
        };
        registry.register_model(record).expect("register");
        let queried = registry.query_model("gpt-4o-mini").expect("query");
        assert_eq!(queried.version, 1);
    }
}
