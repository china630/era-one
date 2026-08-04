use era_office_richtext::{apply_frame_op, FrameKey, FrameOp, TextFrame};
use serde::{Deserialize, Deserializer, Serialize};
use serde_json::Value;

#[derive(Debug, Clone, Serialize, PartialEq)]
pub struct ErapSlide {
    pub id: String,
    pub title_frame: TextFrame,
    pub body_frame: TextFrame,
    #[serde(default)]
    pub body2_frame: TextFrame,
    #[serde(default = "default_layout")]
    pub layout: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub background: Option<String>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub notes: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub image_url: Option<String>,
    /// Slide transition: none | fade | push | wipe | morph (lite CSS in UI).
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub transition: String,
    /// Object animation: none | appear (stagger in Present).
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub animation: String,
}

fn default_layout() -> String {
    "title_body".into()
}

impl ErapSlide {
    pub fn new_blank() -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            title_frame: TextFrame::from_plain("Title"),
            body_frame: TextFrame::empty(),
            body2_frame: TextFrame::empty(),
            layout: default_layout(),
            background: None,
            notes: String::new(),
            image_url: None,
            transition: String::new(),
            animation: String::new(),
        }
    }

    pub fn title(&self) -> String {
        self.title_frame.plain_text()
    }

    pub fn body(&self) -> String {
        self.body_frame.plain_text()
    }

    pub fn body2(&self) -> String {
        self.body2_frame.plain_text()
    }

    pub fn set_title_plain(&mut self, s: impl Into<String>) {
        self.title_frame = TextFrame::from_plain(s);
    }

    pub fn set_body_plain(&mut self, s: impl Into<String>) {
        self.body_frame = TextFrame::from_plain(s);
    }

    pub fn frame_mut(&mut self, key: FrameKey) -> &mut TextFrame {
        match key {
            FrameKey::Title => &mut self.title_frame,
            FrameKey::Body => &mut self.body_frame,
            FrameKey::Body2 => &mut self.body2_frame,
        }
    }

    pub fn apply_op(&mut self, key: FrameKey, op: &FrameOp) {
        apply_frame_op(self.frame_mut(key), op);
    }
}

impl<'de> Deserialize<'de> for ErapSlide {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let v = Value::deserialize(deserializer)?;
        let obj = v
            .as_object()
            .ok_or_else(|| serde::de::Error::custom("slide must be object"))?;

        let id = obj
            .get("id")
            .and_then(|x| x.as_str())
            .unwrap_or("")
            .to_string();
        let layout = obj
            .get("layout")
            .and_then(|x| x.as_str())
            .unwrap_or("title_body")
            .to_string();
        let background = obj
            .get("background")
            .and_then(|x| x.as_str())
            .map(|s| s.to_string());
        let notes = obj
            .get("notes")
            .and_then(|x| x.as_str())
            .unwrap_or("")
            .to_string();
        let image_url = obj
            .get("image_url")
            .and_then(|x| x.as_str())
            .map(|s| s.to_string());
        let transition = obj
            .get("transition")
            .and_then(|x| x.as_str())
            .unwrap_or("")
            .to_string();
        let animation = obj
            .get("animation")
            .and_then(|x| x.as_str())
            .unwrap_or("")
            .to_string();

        let mut title_frame =
            parse_frame(obj, "title_frame", "title").map_err(serde::de::Error::custom)?;
        let mut body_frame =
            parse_frame(obj, "body_frame", "body").map_err(serde::de::Error::custom)?;
        let mut body2_frame =
            parse_frame(obj, "body2_frame", "body2").map_err(serde::de::Error::custom)?;

        // Migrate legacy whole-field format flags into span/block attrs.
        apply_legacy_chrome(
            &mut title_frame,
            obj,
            "title_bold",
            "title_align",
            "title_font",
            "title_font_pt",
        );
        apply_legacy_chrome(
            &mut body_frame,
            obj,
            "body_bold",
            "body_align",
            "body_font",
            "body_font_pt",
        );
        apply_legacy_chrome(
            &mut body2_frame,
            obj,
            "body_bold",
            "body_align",
            "body_font",
            "body_font_pt",
        );

        Ok(ErapSlide {
            id: if id.is_empty() {
                uuid::Uuid::new_v4().to_string()
            } else {
                id
            },
            title_frame,
            body_frame,
            body2_frame,
            layout,
            background,
            notes,
            image_url,
            transition,
            animation,
        })
    }
}

fn parse_frame(
    obj: &serde_json::Map<String, Value>,
    frame_key: &str,
    legacy_key: &str,
) -> Result<TextFrame, String> {
    if let Some(v) = obj.get(frame_key) {
        return serde_json::from_value(v.clone()).map_err(|e| e.to_string());
    }
    if let Some(v) = obj.get(legacy_key) {
        if let Some(s) = v.as_str() {
            return Ok(TextFrame::from_plain(s));
        }
        return serde_json::from_value(v.clone()).map_err(|e| e.to_string());
    }
    Ok(TextFrame::empty())
}

fn apply_legacy_chrome(
    frame: &mut TextFrame,
    obj: &serde_json::Map<String, Value>,
    bold_k: &str,
    align_k: &str,
    font_k: &str,
    pt_k: &str,
) {
    frame.ensure_nonempty();
    let bold = obj.get(bold_k).and_then(|x| x.as_bool()).unwrap_or(false);
    let align = obj.get(align_k).and_then(|x| x.as_str());
    let font = obj.get(font_k).and_then(|x| x.as_str());
    let pt = obj.get(pt_k).and_then(|x| x.as_u64()).map(|n| n as u32);
    if !bold && align.is_none() && font.is_none() && pt.is_none() {
        return;
    }
    for b in &mut frame.blocks {
        if let Some(a) = align {
            b.align = match a {
                "center" => era_office_richtext::TextAlign::Center,
                "right" => era_office_richtext::TextAlign::Right,
                "justify" => era_office_richtext::TextAlign::Justify,
                _ => era_office_richtext::TextAlign::Left,
            };
        }
        for sp in &mut b.inlines {
            if bold {
                sp.bold = true;
            }
            if let Some(f) = font {
                sp.font_family = Some(f.to_string());
            }
            if let Some(p) = pt {
                sp.font_size_pt = Some(p);
            }
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ErapDeck {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub tenant_id: String,
    #[serde(default)]
    pub drive_object_id: String,
    #[serde(default = "fmt")]
    pub format: String,
    pub name: String,
    pub slides: Vec<ErapSlide>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub theme_background: Option<String>,
    /// P-LITE: default layout for new slides.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub default_layout: Option<String>,
    /// P-LITE: placeholder plain text for new slide title.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub master_title_placeholder: Option<String>,
    /// P-LITE: placeholder plain text for new slide body.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub master_body_placeholder: Option<String>,
    /// Optimistic concurrency for frame-ops (lab).
    #[serde(default)]
    pub version: u64,
}

fn fmt() -> String {
    "erap".into()
}

impl ErapDeck {
    pub fn empty() -> Self {
        Self {
            id: String::new(),
            tenant_id: String::new(),
            drive_object_id: String::new(),
            format: "erap".into(),
            name: "deck.erap".into(),
            slides: vec![ErapSlide::new_blank()],
            theme_background: None,
            default_layout: None,
            master_title_placeholder: None,
            master_body_placeholder: None,
            version: 0,
        }
    }

    /// Apply master placeholders / layout when creating a blank slide.
    pub fn new_slide_from_master(&self) -> ErapSlide {
        let mut s = ErapSlide::new_blank();
        if let Some(layout) = &self.default_layout {
            if !layout.is_empty() {
                s.layout = layout.clone();
            }
        }
        if let Some(t) = &self.master_title_placeholder {
            if !t.is_empty() {
                s.set_title_plain(t.clone());
            }
        }
        if let Some(b) = &self.master_body_placeholder {
            s.set_body_plain(b.clone());
        }
        s
    }

    pub fn dump_plain(&self) -> String {
        self.slides
            .iter()
            .map(|s| format!("{}|{}", s.title(), s.body()))
            .collect::<Vec<_>>()
            .join("\n")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn legacy_string_title_body_deserializes() {
        let raw = r#"{"id":"s1","title":"Hi","body":"Line1\nLine2","layout":"title_body"}"#;
        let s: ErapSlide = serde_json::from_str(raw).unwrap();
        assert_eq!(s.title(), "Hi");
        assert_eq!(s.body_frame.blocks.len(), 2);
        assert_eq!(s.body(), "Line1\nLine2");
    }

    #[test]
    fn frame_roundtrip_json() {
        let mut s = ErapSlide::new_blank();
        s.set_title_plain("Agenda");
        s.set_body_plain("Item");
        let json = serde_json::to_string(&s).unwrap();
        assert!(json.contains("title_frame"));
        let back: ErapSlide = serde_json::from_str(&json).unwrap();
        assert_eq!(back.title(), "Agenda");
        assert_eq!(back.body(), "Item");
    }

    #[test]
    fn motion_fields_roundtrip() {
        let raw = r#"{"id":"s1","title":"T","body":"B","transition":"morph","animation":"appear"}"#;
        let s: ErapSlide = serde_json::from_str(raw).unwrap();
        assert_eq!(s.transition, "morph");
        assert_eq!(s.animation, "appear");
        let json = serde_json::to_string(&s).unwrap();
        assert!(json.contains("\"transition\":\"morph\""));
        assert!(json.contains("\"animation\":\"appear\""));
    }

    #[test]
    fn master_fields_persist_and_new_slide() {
        let mut deck = ErapDeck::empty();
        deck.default_layout = Some("two_column".into());
        deck.master_title_placeholder = Some("From master".into());
        deck.master_body_placeholder = Some("Body ph".into());
        deck.theme_background = Some("#112233".into());
        let json = serde_json::to_string(&deck).unwrap();
        assert!(json.contains("default_layout"));
        assert!(json.contains("master_title_placeholder"));
        let back: ErapDeck = serde_json::from_str(&json).unwrap();
        assert_eq!(back.default_layout.as_deref(), Some("two_column"));
        let slide = back.new_slide_from_master();
        assert_eq!(slide.layout, "two_column");
        assert_eq!(slide.title(), "From master");
        assert_eq!(slide.body(), "Body ph");
    }
}
