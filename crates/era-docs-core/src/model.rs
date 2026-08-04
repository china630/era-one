use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum BlockType {
    #[default]
    Paragraph,
    Heading,
    ListItem,
    PageBreak,
    /// Wave W2: image block (url or data URL).
    Image,
    /// Wave W2: simple table (pipe-rows in inlines[0].text as TSV/JSON lite).
    Table,
    /// Wave W2: bookmark anchor.
    Bookmark,
    /// Wave W2: generated table of contents placeholder.
    Toc,
    /// Wave LATER: bordered text box (lite).
    TextBox,
    /// Wave ERA+: section break marker.
    SectionBreak,
    /// Wave ERA+: footnote body (lite; marker in previous para via bookmark_name).
    Footnote,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ListType {
    Bullet,
    Ordered,
}

/// O-FMT-1: limited marker set (not full Word abstractNum).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum ListMarker {
    #[default]
    Disc,
    Circle,
    Square,
    Decimal,
    LowerAlpha,
    LowerRoman,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum TextAlign {
    #[default]
    Left,
    Center,
    Right,
    Justify,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct InlineSpan {
    pub text: String,
    #[serde(default)]
    pub bold: bool,
    #[serde(default)]
    pub italic: bool,
    #[serde(default)]
    pub underline: bool,
    #[serde(default)]
    pub strike: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub link_url: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub font_family: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub font_size_pt: Option<u32>,
    /// CSS color (e.g. #c00) — Wave W2.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub color: Option<String>,
    /// Highlight background — Wave W2.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub highlight: Option<String>,
    /// O-FMT-1: superscript / subscript (docx vertAlign).
    #[serde(default)]
    pub superscript: bool,
    #[serde(default)]
    pub subscript: bool,
}

impl InlineSpan {
    pub fn plain(text: impl Into<String>) -> Self {
        Self {
            text: text.into(),
            bold: false,
            italic: false,
            underline: false,
            strike: false,
            link_url: None,
            font_family: None,
            font_size_pt: None,
            color: None,
            highlight: None,
            superscript: false,
            subscript: false,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Block {
    pub id: String,
    #[serde(default)]
    pub block_type: BlockType,
    #[serde(default)]
    pub heading_level: u32,
    #[serde(default)]
    pub list_type: Option<ListType>,
    #[serde(default)]
    pub align: TextAlign,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub line_spacing: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub indent_mm: Option<u32>,
    /// O-FMT-1: paragraph spacing (points).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub space_before_pt: Option<u32>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub space_after_pt: Option<u32>,
    /// O-FMT-1: list nest level (0 = top); do not reuse heading_level.
    #[serde(default)]
    pub list_level: u32,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub list_marker: Option<ListMarker>,
    /// O-FMT-1: when true, ordered list restarts at this item.
    #[serde(default)]
    pub list_restart: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub style_name: Option<String>,
    /// Image URL / data URL when block_type=Image.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub image_url: Option<String>,
    /// Bookmark name when block_type=Bookmark.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub bookmark_name: Option<String>,
    /// Language tag (BCP-47) — Wave W2.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub lang: Option<String>,
    /// O-LITE P1: structured table cells (rows × columns); else TSV in inlines.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub table_cells: Option<Vec<Vec<String>>>,
    #[serde(default)]
    pub inlines: Vec<InlineSpan>,
}

impl Block {
    pub fn paragraph(id: impl Into<String>, text: impl Into<String>) -> Self {
        Self {
            id: id.into(),
            block_type: BlockType::Paragraph,
            heading_level: 0,
            list_type: None,
            align: TextAlign::Left,
            line_spacing: None,
            indent_mm: None,
            space_before_pt: None,
            space_after_pt: None,
            list_level: 0,
            list_marker: None,
            list_restart: false,
            style_name: None,
            image_url: None,
            bookmark_name: None,
            lang: None,
            table_cells: None,
            inlines: vec![InlineSpan::plain(text)],
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DocComment {
    pub id: String,
    pub block_id: String,
    #[serde(default)]
    pub author_id: String,
    pub text: String,
    #[serde(default)]
    pub resolved: bool,
    /// O-LITE P1: selection range (char offsets in block) + quoted snippet.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub start: Option<usize>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub end: Option<usize>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub quote: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PageSetup {
    #[serde(default = "default_page_size")]
    pub size: String,
    #[serde(default = "default_orientation")]
    pub orientation: String,
    #[serde(default = "default_margins")]
    pub margins_mm: u32,
    /// Wave LATER: CSS column-count (1–3).
    #[serde(default = "default_columns")]
    pub columns: u32,
}

fn default_page_size() -> String {
    "a4".into()
}
fn default_orientation() -> String {
    "portrait".into()
}
fn default_margins() -> u32 {
    20
}
fn default_columns() -> u32 {
    1
}

impl Default for PageSetup {
    fn default() -> Self {
        Self {
            size: default_page_size(),
            orientation: default_orientation(),
            margins_mm: default_margins(),
            columns: default_columns(),
        }
    }
}

/// Wave LATER: track-changes revision (lite).
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DocRevision {
    pub id: String,
    pub block_id: String,
    /// insert | delete | replace
    pub kind: String,
    #[serde(default)]
    pub before: String,
    #[serde(default)]
    pub after: String,
    #[serde(default)]
    pub author_id: String,
}

/// Wave ERA+: named paragraph style definition (manage styles lite).
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct NamedStyle {
    pub name: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub font_family: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub font_size_pt: Option<u32>,
    #[serde(default)]
    pub bold: bool,
    #[serde(default)]
    pub italic: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub color: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, Default)]
pub struct HeaderFooter {
    #[serde(default)]
    pub text: String,
    #[serde(default)]
    pub page_numbers: bool,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct EradDocument {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub tenant_id: String,
    #[serde(default)]
    pub drive_object_id: String,
    #[serde(default = "default_format")]
    pub format: String,
    #[serde(default)]
    pub blocks: Vec<Block>,
    #[serde(default)]
    pub comments: Vec<DocComment>,
    #[serde(default)]
    pub page: PageSetup,
    #[serde(default)]
    pub header: HeaderFooter,
    #[serde(default)]
    pub footer: HeaderFooter,
    #[serde(default)]
    pub track_changes: bool,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub revisions: Vec<DocRevision>,
    /// Wave ERA+: custom named styles (manage styles).
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub styles: Vec<NamedStyle>,
    #[serde(default)]
    pub legacy_features_dropped: bool,
}

fn default_format() -> String {
    "erad".to_string()
}

impl EradDocument {
    pub fn empty() -> Self {
        Self {
            id: String::new(),
            tenant_id: String::new(),
            drive_object_id: String::new(),
            format: "erad".into(),
            blocks: vec![Block::paragraph(uuid::Uuid::new_v4().to_string(), "")],
            comments: vec![],
            page: PageSetup::default(),
            header: HeaderFooter::default(),
            footer: HeaderFooter::default(),
            track_changes: false,
            revisions: vec![],
            styles: vec![],
            legacy_features_dropped: false,
        }
    }

    pub fn plain_text(&self) -> String {
        self.blocks
            .iter()
            .filter(|b| b.block_type != BlockType::PageBreak)
            .map(|b| {
                b.inlines
                    .iter()
                    .map(|s| s.text.as_str())
                    .collect::<String>()
            })
            .collect::<Vec<_>>()
            .join("\n")
    }
}
