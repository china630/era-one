#![no_main]

use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    let _ = era_docs_engine::convert::docx_import::import_docx(data);
});
