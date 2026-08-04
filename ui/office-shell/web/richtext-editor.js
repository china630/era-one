/* Shared block + InlineSpan helpers for Docs / Presentations (ERA Office). */
(function (global) {
  'use strict';

  function spanFromMarks(text, marks, patch) {
    const m = Object.assign({}, marks || {}, patch || {});
    return {
      text: text || '',
      bold: !!m.bold,
      italic: !!m.italic,
      underline: !!m.underline,
      strike: !!m.strike,
      superscript: !!m.superscript,
      subscript: !!m.subscript,
      link_url: m.link_url || undefined,
      font_family: m.font_family || undefined,
      font_size_pt: m.font_size_pt || undefined,
      color: m.color || undefined,
      highlight: m.highlight || undefined,
    };
  }

  function marksEqual(a, b) {
    if (!a || !b) return false;
    return (
      !!a.bold === !!b.bold &&
      !!a.italic === !!b.italic &&
      !!a.underline === !!b.underline &&
      !!a.strike === !!b.strike &&
      !!a.superscript === !!b.superscript &&
      !!a.subscript === !!b.subscript &&
      (a.link_url || '') === (b.link_url || '') &&
      (a.font_family || '') === (b.font_family || '') &&
      (a.font_size_pt || 0) === (b.font_size_pt || 0) &&
      (a.color || '') === (b.color || '') &&
      (a.highlight || '') === (b.highlight || '')
    );
  }

  function coalesceInlines(inlines) {
    const out = [];
    for (const span of inlines || []) {
      if (!span || !span.text) continue;
      const last = out[out.length - 1];
      if (last && marksEqual(last, span)) last.text += span.text;
      else out.push(Object.assign({}, span));
    }
    if (!out.length) out.push(spanFromMarks('', {}));
    return out;
  }

  function blockText(block) {
    return (block.inlines || []).map((i) => i.text || '').join('');
  }

  function framePlain(frame) {
    return (frame.blocks || []).map(blockText).join('\n');
  }

  function splitInlinesAt(inlines, offset) {
    const list = (inlines || []).map((s) => Object.assign({}, s, { text: s.text || '' }));
    const plain = list.map((s) => s.text).join('');
    const off = Math.max(0, Math.min(offset, plain.length));
    if (off === 0) return { left: [], right: list };
    if (off >= plain.length) return { left: list, right: [] };
    let acc = 0;
    for (let i = 0; i < list.length; i++) {
      const len = list[i].text.length;
      if (acc + len === off) {
        return {
          left: coalesceInlines(list.slice(0, i + 1)),
          right: coalesceInlines(list.slice(i + 1)),
        };
      }
      if (acc + len > off) {
        const local = off - acc;
        const leftSpan = spanFromMarks(list[i].text.slice(0, local), list[i]);
        const rightSpan = spanFromMarks(list[i].text.slice(local), list[i]);
        const left = list.slice(0, i).concat(leftSpan.text ? [leftSpan] : []);
        const right = (rightSpan.text ? [rightSpan] : []).concat(list.slice(i + 1));
        return { left: coalesceInlines(left), right: coalesceInlines(right) };
      }
      acc += len;
    }
    return { left: list, right: [] };
  }

  function insertTextPreserving(inlines, offset, text, typing) {
    if (!text) return coalesceInlines(inlines);
    const { left, right } = splitInlinesAt(inlines, offset);
    const template =
      typing ||
      (left.length ? left[left.length - 1] : null) ||
      (right.length ? right[0] : null) ||
      {};
    return coalesceInlines(left.concat([spanFromMarks(text, template)], right));
  }

  function deleteRangePreserving(inlines, start, end) {
    if (start >= end) return coalesceInlines(inlines);
    const a = splitInlinesAt(inlines, start);
    const b = splitInlinesAt(a.right, end - start);
    return coalesceInlines(a.left.concat(b.right));
  }

  function applyMarksRangeLocal(inlines, start, end, patch) {
    if (start >= end) return coalesceInlines(inlines);
    const head = splitInlinesAt(inlines, start);
    const midRight = splitInlinesAt(head.right, end - start);
    const mid = midRight.left.map((s) => spanFromMarks(s.text, s, patch));
    return coalesceInlines(head.left.concat(mid, midRight.right));
  }

  function diffText(prev, next) {
    let start = 0;
    const minLen = Math.min(prev.length, next.length);
    while (start < minLen && prev.charAt(start) === next.charAt(start)) start++;
    let endPrev = prev.length;
    let endNext = next.length;
    while (
      endPrev > start &&
      endNext > start &&
      prev.charAt(endPrev - 1) === next.charAt(endNext - 1)
    ) {
      endPrev--;
      endNext--;
    }
    return { start, deletedEnd: endPrev, inserted: next.slice(start, endNext) };
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function renderInlineHtml(block) {
    const spans = block.inlines || [];
    if (!spans.length) return '';
    return spans
      .map((span) => {
        let html = escapeHtml(span.text || '').replace(/\n/g, '<br>');
        if (span.bold) html = '<b>' + html + '</b>';
        if (span.italic) html = '<i>' + html + '</i>';
        if (span.underline) html = '<u>' + html + '</u>';
        if (span.strike) html = '<s>' + html + '</s>';
        const style = [];
        if (span.font_family) style.push('font-family:' + span.font_family);
        if (span.font_size_pt) style.push('font-size:' + span.font_size_pt + 'pt');
        if (span.color) style.push('color:' + span.color);
        if (span.highlight) style.push('background:' + span.highlight);
        if (style.length) html = '<span style="' + style.join(';') + '">' + html + '</span>';
        return html;
      })
      .join('');
  }

  function selectionOffsetsInBlock(blockEl) {
    const sel = window.getSelection();
    if (!sel || !blockEl) return null;
    function offsetFromPoint(node, nodeOffset) {
      const range = document.createRange();
      range.selectNodeContents(blockEl);
      range.setEnd(node, nodeOffset);
      return range.toString().length;
    }
    try {
      if (!blockEl.contains(sel.anchorNode)) return null;
      const a = offsetFromPoint(sel.anchorNode, sel.anchorOffset);
      const f = blockEl.contains(sel.focusNode)
        ? offsetFromPoint(sel.focusNode, sel.focusOffset)
        : a;
      return { start: Math.min(a, f), end: Math.max(a, f) };
    } catch (_) {
      return null;
    }
  }

  function newId(prefix) {
    return (prefix || 'b') + '-' + (crypto.randomUUID ? crypto.randomUUID() : Date.now());
  }

  function emptyFrame(text) {
    return {
      blocks: [
        {
          id: newId('b'),
          block_type: 'paragraph',
          align: 'left',
          inlines: [spanFromMarks(text || '', {})],
        },
      ],
    };
  }

  function copyBlockShell(block, id, inlines) {
    return {
      id,
      block_type: block.block_type || 'paragraph',
      heading_level: block.heading_level || 0,
      list_type: block.list_type,
      align: block.align || 'left',
      list_level: block.list_level || 0,
      list_marker: block.list_marker,
      style_name: block.style_name,
      inlines: coalesceInlines(inlines),
    };
  }

  global.EraRichText = {
    spanFromMarks,
    coalesceInlines,
    blockText,
    framePlain,
    splitInlinesAt,
    insertTextPreserving,
    deleteRangePreserving,
    applyMarksRangeLocal,
    diffText,
    escapeHtml,
    renderInlineHtml,
    selectionOffsetsInBlock,
    newId,
    emptyFrame,
    copyBlockShell,
  };
})(typeof window !== 'undefined' ? window : globalThis);
