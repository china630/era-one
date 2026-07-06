//! Modbus/TCP ADU parser — function 3 (Read Holding Registers) stub.

/// Разобранный Modbus/TCP кадр.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ModbusFrame {
    pub transaction_id: u16,
    pub unit_id: u8,
    pub function_code: u8,
    pub register_count: u16,
}

/// Парсит минимальный Modbus/TCP PDU (MBAP header + function 3).
/// Возвращает ошибку при коротком буфере или неизвестной function.
pub fn parse_modbus_frame(raw: &[u8]) -> Result<ModbusFrame, &'static str> {
    if raw.len() < 12 {
        return Err("frame too short");
    }
    let transaction_id = u16::from_be_bytes([raw[0], raw[1]]);
    let protocol_id = u16::from_be_bytes([raw[2], raw[3]]);
    if protocol_id != 0 {
        return Err("not modbus/tcp");
    }
    let length = u16::from_be_bytes([raw[4], raw[5]]) as usize;
    if length + 6 != raw.len() {
        return Err("length mismatch");
    }
    let unit_id = raw[6];
    let function_code = raw[7];
    if function_code != 3 {
        return Err("unsupported function");
    }
    if raw.len() < 12 {
        return Err("missing register fields");
    }
    let register_count = u16::from_be_bytes([raw[10], raw[11]]);
    Ok(ModbusFrame {
        transaction_id,
        unit_id,
        function_code,
        register_count,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::path::PathBuf;

    #[test]
    fn modbus_golden() {
        let dir = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("testdata");
        let raw = fs::read(dir.join("modbus_read_holding.bin")).expect("read sample");
        let got = parse_modbus_frame(&raw).expect("parse");
        let want_tx = u16::from_be_bytes([0x12, 0x34]);
        assert_eq!(got.transaction_id, want_tx);
        assert_eq!(got.unit_id, 1);
        assert_eq!(got.function_code, 3);
        assert_eq!(got.register_count, 2);
        let golden = fs::read_to_string(dir.join("modbus_frame.golden.txt")).expect("golden");
        let summary = format!(
            "tx={} unit={} fn={} regs={}",
            got.transaction_id, got.unit_id, got.function_code, got.register_count
        );
        assert_eq!(summary.trim(), golden.trim());
    }
}
