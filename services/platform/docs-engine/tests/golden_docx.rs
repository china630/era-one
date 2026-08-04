use era_docs_engine::canonical::{
    from_canonical_json, structure_equiv, strip_volatile_ids, to_canonical_json,
};
use era_docs_engine::convert::docx_import::{import_docx, minimal_docx};
use era_docs_engine::convert::export_docx;

#[test]
fn golden_docx_az_memo_roundtrip() {
    let docx = minimal_docx(&["Hello memo", "Second line"]).expect("docx");
    let imported = import_docx(&docx).expect("import");
    let golden_path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("testdata/az_memo_roundtrip.golden.erad.json");

    if std::env::var("UPDATE_GOLDEN").ok().as_deref() == Some("1") {
        let json = to_canonical_json(&strip_volatile_ids(imported.clone())).expect("json");
        std::fs::write(&golden_path, json).expect("write golden");
    }

    assert!(
        golden_path.exists(),
        "missing golden {}",
        golden_path.display()
    );
    let want = std::fs::read_to_string(&golden_path).expect("read golden");
    let want_doc = from_canonical_json(&want).expect("parse golden");
    assert!(
        structure_equiv(&imported, &want_doc),
        "structure mismatch vs golden; got plain={:?} want={:?}",
        imported.plain_text(),
        want_doc.plain_text()
    );

    let exported = export_docx(&imported).expect("export");
    let reimported = import_docx(&exported).expect("reimport");
    assert_eq!(imported.plain_text(), reimported.plain_text());
}
