//! Кольцевой буфер событий (backpressure, ADR-0008).

use crate::Envelope;
use std::collections::VecDeque;

pub struct RingBuffer {
    capacity: usize,
    items: VecDeque<Envelope>,
    dropped: u64,
}

impl RingBuffer {
    pub fn new(capacity: usize) -> Self {
        Self {
            capacity: capacity.max(1),
            items: VecDeque::with_capacity(capacity.max(1)),
            dropped: 0,
        }
    }

    pub fn push(&mut self, env: Envelope) {
        if self.items.len() >= self.capacity {
            self.items.pop_front();
            self.dropped += 1;
        }
        self.items.push_back(env);
    }

    pub fn drain(&mut self, n: usize) -> Vec<Envelope> {
        let take = n.min(self.items.len());
        self.items.drain(..take).collect()
    }

    /// Возвращает события в начало буфера (backpressure RETRY/THROTTLE).
    pub fn requeue_front(&mut self, batch: Vec<Envelope>) {
        if batch.is_empty() {
            return;
        }
        while self.items.len() + batch.len() > self.capacity {
            self.items.pop_front();
            self.dropped += 1;
        }
        let mut front = VecDeque::from(batch);
        front.append(&mut self.items);
        self.items = front;
    }

    pub fn len(&self) -> usize {
        self.items.len()
    }

    #[allow(dead_code)]
    pub fn is_empty(&self) -> bool {
        self.items.is_empty()
    }

    pub fn dropped(&self) -> u64 {
        self.dropped
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sample;

    #[test]
    fn drops_oldest_on_overflow() {
        let mut b = RingBuffer::new(2);
        b.push(sample::process_envelope("a"));
        b.push(sample::process_envelope("b"));
        b.push(sample::process_envelope("c"));
        assert_eq!(b.len(), 2);
        assert_eq!(b.dropped(), 1);
    }

    #[test]
    fn requeue_front_preserves_order() {
        let mut b = RingBuffer::new(10);
        b.push(sample::process_envelope("a"));
        b.requeue_front(vec![
            sample::process_envelope("x"),
            sample::process_envelope("y"),
        ]);
        assert_eq!(b.len(), 3);
    }
}
