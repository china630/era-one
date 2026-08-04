use serde::{Deserialize, Deserializer, Serialize};

use crate::model::{Block, InlineSpan};
use crate::spans::block_plain;

/// A text shape: ordered paragraphs (`Block`s) with character runs.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct TextFrame {
    pub blocks: Vec<Block>,
}

impl Default for TextFrame {
    fn default() -> Self {
        Self::empty()
    }
}

impl TextFrame {
    pub fn empty() -> Self {
        Self {
            blocks: vec![Block::paragraph(uuid::Uuid::new_v4().to_string(), "")],
        }
    }

    pub fn from_plain(text: impl Into<String>) -> Self {
        let text = text.into();
        // Split on newlines into paragraphs (import/legacy convenience).
        let parts: Vec<&str> = if text.is_empty() {
            vec![""]
        } else {
            text.split('\n').collect()
        };
        Self {
            blocks: parts
                .into_iter()
                .map(|p| Block::paragraph(uuid::Uuid::new_v4().to_string(), p))
                .collect(),
        }
    }

    pub fn plain_text(&self) -> String {
        self.blocks
            .iter()
            .map(|b| block_plain(&b.inlines))
            .collect::<Vec<_>>()
            .join("\n")
    }

    pub fn ensure_nonempty(&mut self) {
        if self.blocks.is_empty() {
            *self = Self::empty();
        }
    }
}

impl<'de> Deserialize<'de> for TextFrame {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        #[derive(Deserialize)]
        #[serde(untagged)]
        enum Wire {
            Frame { blocks: Vec<Block> },
            Plain(String),
            /// Legacy accidental shape: array of spans as single paragraph.
            Spans(Vec<InlineSpan>),
        }
        match Wire::deserialize(deserializer)? {
            Wire::Frame { blocks } => {
                let mut f = TextFrame { blocks };
                f.ensure_nonempty();
                Ok(f)
            }
            Wire::Plain(s) => Ok(TextFrame::from_plain(s)),
            Wire::Spans(inlines) => Ok(TextFrame {
                blocks: vec![Block {
                    id: uuid::Uuid::new_v4().to_string(),
                    inlines: if inlines.is_empty() {
                        vec![InlineSpan::plain("")]
                    } else {
                        inlines
                    },
                    ..Block::paragraph("", "")
                }],
            }),
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FrameKey {
    Title,
    Body,
    Body2,
}
