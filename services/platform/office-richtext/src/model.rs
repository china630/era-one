use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum BlockType {
    #[default]
    Paragraph,
    Heading,
    ListItem,
    PageBreak,
    Image,
    Table,
    Bookmark,
    Toc,
    TextBox,
    SectionBreak,
    Footnote,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ListType {
    Bullet,
    Ordered,
}

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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub color: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub highlight: Option<String>,
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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub space_before_pt: Option<u32>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub space_after_pt: Option<u32>,
    #[serde(default)]
    pub list_level: u32,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub list_marker: Option<ListMarker>,
    #[serde(default)]
    pub list_restart: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub style_name: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub image_url: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub bookmark_name: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub lang: Option<String>,
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
            inlines: vec![InlineSpan::plain(text)],
        }
    }
}
