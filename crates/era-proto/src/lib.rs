//! Сгенерированные типы ERA XDR из `proto/era/v1/`.
//!
//! Перегенерация: `scripts/gen-proto.ps1` или `cargo build -p era-proto`.
//! Источник истины — `.proto`; этот крейт не редактируется вручную.

pub mod era {
    pub mod v1 {
        tonic::include_proto!("era.v1");
    }
}

pub use era::v1::*;

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn envelope_roundtrip_prost() {
        let env = Envelope {
            schema_version: "1.0.0".into(),
            event_id: b"0123456789abcdef".to_vec(),
            category: EventCategory::Process as i32,
            severity: Severity::Medium as i32,
            pii_sanitized: true,
            ..Default::default()
        };
        let encoded = prost::Message::encode_to_vec(&env);
        let decoded = Envelope::decode(encoded.as_slice()).expect("decode envelope");
        assert_eq!(decoded.schema_version, "1.0.0");
        assert!(decoded.pii_sanitized);
    }
}
