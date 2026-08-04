use era_docs_engine::wire::{Block, BlockAttrs, BlockType, DocumentFormat, EradDocument as WireDoc, InlineSpan};

#[test]
fn erad_wire_proto_roundtrip() {
    let doc = WireDoc {
        id: "d1".into(),
        tenant_id: "t1".into(),
        drive_object_id: "o1".into(),
        format: DocumentFormat::Erad as i32,
        blocks: vec![Block {
            id: "b1".into(),
            attrs: Some(BlockAttrs {
                block_type: BlockType::Paragraph as i32,
                heading_level: 0,
                list_type: 0,
                list_level: 0,
                space_before_pt: 0,
                space_after_pt: 0,
                line_spacing: String::new(),
                indent_mm: 0,
                style_name: String::new(),
            }),
            inlines: vec![InlineSpan {
                text: "Hello".into(),
                bold: false,
                italic: false,
                underline: false,
                link_url: String::new(),
                strike: false,
                superscript: false,
                subscript: false,
                font_family: String::new(),
                font_size_pt: 0,
                color: String::new(),
                highlight: String::new(),
            }],
        }],
        legacy_features_dropped: false,
        comments: vec![],
    };
    let encoded = prost::Message::encode_to_vec(&doc);
    assert!(!encoded.is_empty());
    let golden = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("testdata/erad_minimal.golden.bin");
    if std::env::var("UPDATE_GOLDEN").ok().as_deref() == Some("1") {
        std::fs::write(&golden, &encoded).expect("write golden bin");
    }
    let want = std::fs::read(&golden).expect("read golden bin");
    let decoded: WireDoc = prost::Message::decode(want.as_slice()).expect("decode");
    assert_eq!(decoded.plain_text(), "Hello");
}

trait Plain {
    fn plain_text(&self) -> String;
}

impl Plain for WireDoc {
    fn plain_text(&self) -> String {
        self.blocks
            .iter()
            .flat_map(|b| b.inlines.iter().map(|s| s.text.as_str()))
            .collect::<Vec<_>>()
            .join("\n")
    }
}
