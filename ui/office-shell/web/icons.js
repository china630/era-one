/* Shared ERA Office icon set (Wave G). Own SVG paths — not Google Material. */
(function (global) {
  var ICONS = {
    bold:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M7 4h6.5a4.5 4.5 0 0 1 3.2 7.7A4.8 4.8 0 0 1 13.2 20H7V4zm3 6.2h3.1a1.8 1.8 0 0 0 0-3.6H10v3.6zm0 6.6h3.6a2.1 2.1 0 0 0 0-4.2H10v4.2z"/></svg>',
    italic:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M10 4h9v3h-3.2l-4.1 10H15v3H6v-3h3.2l4.1-10H10V4z"/></svg>',
    underline:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M6 3h3v9.2a3 3 0 0 0 6 0V3h3v9.2a6 6 0 0 1-12 0V3zM5 20h14v2H5v-2z"/></svg>',
    strike:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M7.5 11h9v2h-9v-2zm1.2-6C11 4 14 3.8 16 5.2l-1.5 2.2c-1-.7-2.5-.8-3.7-.2-.7.4-.9 1-.9 1.5H7.2C7.2 6.8 8.2 5.2 8.7 5zm6.6 9.5c.3.5.4 1.1 0 1.7-.7 1.1-2.5 1.5-4.1 1.1-1.2-.3-2-.9-2.5-1.5l1.7-1.9c.4.5 1.1.9 2 .9.7 0 1.1-.2 1.3-.5.2-.2.1-.5-.2-.6l-5-.2v-2.1l7.8.3c1.1.1 2 .7 2.3 1.6z"/></svg>',
    alignLeft:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5h18v2H3V5zm0 4h12v2H3V9zm0 4h18v2H3v-2zm0 4h12v2H3v-2z"/></svg>',
    alignCenter:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5h18v2H3V5zm3 4h12v2H6V9zm-3 4h18v2H3v-2zm3 4h12v2H6v-2z"/></svg>',
    alignRight:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5h18v2H3V5zm6 4h12v2H9V9zm-6 4h18v2H3v-2zm6 4h12v2H9v-2z"/></svg>',
    alignJustify:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5h18v2H3V5zm0 4h18v2H3V9zm0 4h18v2H3v-2zm0 4h18v2H3v-2z"/></svg>',
    indentDec:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5h18v2H3V5zm8 4h10v2H11V9zm0 4h10v2H11v-2zM3 17h18v2H3v-2zM9 10.5 5 12.5l4 2V10.5z"/></svg>',
    indentInc:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5h18v2H3V5zm0 4h10v2H3V9zm0 4h10v2H3v-2zm0 4h18v2H3v-2zm12-7.5v4l4-2-4-2z"/></svg>',
    formatPainter:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M6 3h8v4h4v4.2l-2 1.2V21H8v-8.6L6 11.2V3zm2 2v4h8V9h-2V5H8zm2 8v6h4v-6h-4z"/></svg>',
    superscript:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M5 16.5 10.2 7h2.3L7.3 16.5H5zm9.2 0L19.5 7H21.8l-5.3 9.5h-2.3zM16 4h5v1.4c0 .8-.3 1.3-.9 1.7l-1.4.9c-.3.2-.4.4-.4.7H20V10h-4V8.7c0-.9.3-1.5 1-1.9l1.5-.9c.2-.1.3-.3.3-.5H16V4z"/></svg>',
    subscript:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M5 16.5 10.2 7h2.3L7.3 16.5H5zm9.2 0L19.5 7H21.8l-5.3 9.5h-2.3zM16 14h5v1.4c0 .8-.3 1.3-.9 1.7l-1.4.9c-.3.2-.4.4-.4.7H20V20h-4v-1.3c0-.9.3-1.5 1-1.9l1.5-.9c.2-.1.3-.3.3-.5H16V14z"/></svg>',
    wrap:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 5h16v2H4V5zm0 4h12a4 4 0 0 1 0 8H9v2.5L5 16l4-3.5V15h7a2 2 0 0 0 0-4H4V9zm0 10h6v2H4v-2z"/></svg>',
    border:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 4h16v16H4V4zm2 2v12h12V6H6zm2 2h8v2H8V8zm0 4h8v2H8v-2zm0 4h5v2H8v-2z"/></svg>',
    /* Literal + / − (toolbar font-size stepper — not “A↑/A↓”). */
    fontInc:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 4h2v7h7v2h-7v7h-2v-7H4v-2h7V4z"/></svg>',
    fontDec:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M5 11h14v2H5z"/></svg>',
    listLevelDec:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 6h16v2H4V6zm6 5h10v2H10v-2zm0 5h10v2H10v-2zM3 11.5 7 9v5l-4-2.5z"/></svg>',
    listLevelInc:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 6h16v2H4V6zm4 5h12v2H8v-2zm0 5h12v2H8v-2zm3-1.5L3 17V12l4 2.5z"/></svg>',
    viewGrid:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 4h7v7H4V4zm9 0h7v7h-7V4zM4 13h7v7H4v-7zm9 0h7v7h-7v-7z"/></svg>',
    viewList:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 5h4v4H4V5zm6 1h10v2H10V6zM4 11h4v4H4v-4zm6 1h10v2H10v-2zM4 17h4v4H4v-4zm6 1h10v2H10v-2z"/></svg>',
    list:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M8 5h13v2H8V5zm0 6h13v2H8v-2zm0 6h13v2H8v-2zM3 6.5A1.5 1.5 0 1 0 3 3.5 1.5 1.5 0 0 0 3 6.5zm0 6A1.5 1.5 0 1 0 3 9.5 1.5 1.5 0 0 0 3 12.5zm0 6A1.5 1.5 0 1 0 3 15.5 1.5 1.5 0 0 0 3 18.5z"/></svg>',
    numbered:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M8 5h13v2H8V5zm0 6h13v2H8v-2zm0 6h13v2H8v-2zM3.5 4H5v5H3.5V5.8H2.2V4.5h1.3V4zm-.3 8.2h2.6v1.2H4.5l1.3 1.4V16H2.8v-1.2h1.5l-1.3-1.4v-1.2zm.5 5.3h1.5V17h1.3v3.5H3.7V19h1.3v-.5H3.7v-1z"/></svg>',
    link:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M10.6 13.4a4 4 0 0 1 0-5.6l2.1-2.1a4 4 0 0 1 5.6 5.6l-1 1-1.4-1.4 1-1a2 2 0 1 0-2.8-2.8l-2.1 2.1a2 2 0 0 0 0 2.8l1.4 1.4-1.8 1zm2.8-2.8a4 4 0 0 1 0 5.6l-2.1 2.1a4 4 0 1 1-5.6-5.6l1-1 1.4 1.4-1 1a2 2 0 1 0 2.8 2.8l2.1-2.1a2 2 0 0 0 0-2.8l-1.4-1.4 1.8-1z"/></svg>',
    comment:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 4h16a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H8l-4 3v-3H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2zm2 4v2h12V8H6zm0 4v2h8v-2H6z"/></svg>',
    save:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M5 3h12l4 4v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2zm2 2v6h10V5H7zm5 9a3 3 0 1 0 .001 6.001A3 3 0 0 0 12 14z"/></svg>',
    share:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M18 8a3 3 0 1 0-2.8-4H15a3 3 0 0 0 0 2.2l-6.1 3.05A3 3 0 0 0 6 9a3 3 0 1 0 0 6 3 3 0 0 0 2.9-2.25L15 15.8A3 3 0 1 0 18 14a3 3 0 0 0-1.1.2l-6.1-3.05c.1-.3.2-.6.2-.95s-.1-.65-.2-.95L16.9 6.2A3 3 0 0 0 18 8z"/></svg>',
    present:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 4h18v12H3V4zm2 2v8h14V6H5zm5 3 5 3-5 3V9zm-7 11h18v2H3v-2z"/></svg>',
    search:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M10 3a7 7 0 1 1 4.3 12.5l4.1 4.1-1.4 1.4-4.1-4.1A7 7 0 0 1 10 3zm0 2a5 5 0 1 0 .001 10.001A5 5 0 0 0 10 5z"/></svg>',
    spell:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 4h9.2l.5 1.5H16v2h-1.6L12.2 18H15v2H7.5l.4-2H5.8v-2h1.9L9.9 7.5H8V5.5h3.1L12 4H4zm6.2 3.5-1.7 8.5h1.9l1.7-8.5h-1.9zM17.2 14.2 19 12.4l1.8 1.8 1.2-1.2-1.8-1.8 1.8-1.8-1.2-1.2-1.8 1.8-1.8-1.8-1.2 1.2 1.8 1.8-1.8 1.8 1.2 1.2z"/></svg>',
    add:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 5h2v6h6v2h-6v6h-2v-6H5v-2h6V5z"/></svg>',
    undo:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 5V2L7 7l5 5V9c3.3 0 6 2.7 6 6a5.9 5.9 0 0 1-1.2 3.5l1.5 1.5A8 8 0 0 0 20 15c0-4.4-3.6-8-8-8z"/></svg>',
    redo:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 5V2l5 5-5 5V9c-3.3 0-6 2.7-6 6a5.9 5.9 0 0 0 1.2 3.5L5.7 20A8 8 0 0 1 4 15c0-4.4 3.6-8 8-8z"/></svg>',
    cut:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M9.6 3.5a3 3 0 1 0-1.9 5.3L10 11l-2.3 2.2A3 3 0 1 0 9.6 20.5L14 16l4.4 4.5 1.4-1.4L15.4 14.5 19.8 10l-1.4-1.4L14 13l-2.5-2.5 2.3-2.2A3 3 0 0 0 9.6 3.5zM7 6a1 1 0 1 1 2 0A1 1 0 0 1 7 6zm0 12a1 1 0 1 1 2 0 1 1 0 0 1-2 0z"/></svg>',
    copy:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M8 4h10a2 2 0 0 1 2 2v12h-2V6H8V4zm-3 4h10a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V10a2 2 0 0 1 2-2zm0 2v10h10V10H5z"/></svg>',
    paste:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M9 3h6v2h3a2 2 0 0 1 2 2v13a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h3V3zm2 2v2h2V5h-2zm-3 4v11h12V9H8z"/></svg>',
    folder:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 6a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6zm2 0v12h14V8h-7.2l-2-2H5z"/></svg>',
    file:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M6 2h8l6 6v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 1.5V9h5.5L14 3.5z"/></svg>',
    download:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 3v10.2l3.5-3.5 1.4 1.4L12 17.5 7.1 11.1l1.4-1.4L12 13.2V3h0zm-7 14h14v2H5v-2z"/></svg>',
    upload:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 17V6.8L8.5 10.3 7.1 8.9 12 4l4.9 4.9-1.4 1.4L12 6.8V17zm-7 2h14v2H5v-2z"/></svg>',
    print:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M7 3h10v4H7V3zm-2 6h14a2 2 0 0 1 2 2v6h-4v4H7v-4H3v-6a2 2 0 0 1 2-2zm2 8v2h8v-2H7zm10-6H7v2h10v-2z"/></svg>',
    settings:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 3h2l.4 2.1a7 7 0 0 1 1.6.9l2-1.1 1.4 1.4-1.1 2c.4.5.7 1 .9 1.6L20 11v2l-2.1.4a7 7 0 0 1-.9 1.6l1.1 2-1.4 1.4-2-1.1a7 7 0 0 1-1.6.9L13 21h-2l-.4-2.1a7 7 0 0 1-1.6-.9l-2 1.1-1.4-1.4 1.1-2a7 7 0 0 1-.9-1.6L4 13v-2l2.1-.4a7 7 0 0 1 .9-1.6l-1.1-2L7.3 5.6l2 1.1a7 7 0 0 1 1.6-.9L11 3zm1 6a3 3 0 1 0 0 6 3 3 0 0 0 0-6z"/></svg>',
    refresh:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M17.7 6.3A8 8 0 1 0 20 12h-2a6 6 0 1 1-1.8-4.2L13 11h7V4l-2.3 2.3z"/></svg>',
    trash:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M9 3h6l1 2h4v2H4V5h4l1-2zm1 6h2v9h-2V9zm4 0h2v9h-2V9zM7 9h2v9H7V9zm-1 11h12v2H6v-2z"/></svg>',
    move:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3.5 6.5A2.5 2.5 0 0 1 6 4h4.2l1.5 1.8H18a2.5 2.5 0 0 1 2.5 2.5V18A2.5 2.5 0 0 1 18 20.5H6A2.5 2.5 0 0 1 3.5 18V6.5zm2 0V18A.5.5 0 0 0 6 18.5h12a.5.5 0 0 0 .5-.5V8.3a.5.5 0 0 0-.5-.5h-6.1L10.4 6H6a.5.5 0 0 0-.5.5z"/><path fill="currentColor" d="M12 10.2v5.1l1.8-1.8 1.1 1.1-3.7 3.7-3.7-3.7 1.1-1.1 1.8 1.8v-5.1h1.6z"/></svg>',
    rename:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 17.2V20h2.8l9.8-9.8-2.8-2.8L4 17.2zm14.7-8.5 1.4-1.4a1 1 0 0 0 0-1.4l-2-2a1 1 0 0 0-1.4 0l-1.4 1.4 2.8 2.8z"/></svg>',
    lock:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a5 5 0 0 1 5 5v3h1a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h1V7a5 5 0 0 1 5-5zm0 2a3 3 0 0 0-3 3v3h6V7a3 3 0 0 0-3-3zm0 9a1.5 1.5 0 0 0-.5 2.9V18h1v-2.1A1.5 1.5 0 0 0 12 13z"/></svg>',
    unlock:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a5 5 0 0 1 5 5h-2a3 3 0 0 0-6 0v3h9a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h1V7a5 5 0 0 1 5-5zm0 11a1.5 1.5 0 0 0-.5 2.9V18h1v-2.1A1.5 1.5 0 0 0 12 13z"/></svg>',
    history:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 4a8 8 0 1 1-8 8H2l3.5 3.5L9 12H6a6 6 0 1 0 6-6V4zm1 4v5l4 2-.8 1.6L11 14V8h2z"/></svg>',
    table:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 4h16v16H4V4zm2 2v3h5V6H6zm7 0v3h5V6h-5zM6 11v3h5v-3H6zm7 0v3h5v-3h-5zM6 16v2h5v-2H6zm7 0v2h5v-2h-5z"/></svg>',
    chart:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 19h16v2H4v-2zm2-2V9h2v8H6zm4 0V5h2v12h-2zm4 0v-6h2v6h-2zm4 0V7h2v10h-2z"/></svg>',
    filter:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 5h16l-6 7.5V19l-4 2v-8.5L4 5z"/></svg>',
    sort:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M8 4h2v12h3l-4 4-4-4h3V4zm6 16h2V8h3l-4-4-4 4h3v12z"/></svg>',
    freeze:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 3h2v4.1l2.9-2.9 1.4 1.4L14.4 8.5H18.5v2h-4.1l2.9 2.9-1.4 1.4L12 11.9l-3.9 3.9-1.4-1.4 2.9-2.9H5.5v-2h4.1L6.7 5.6 8.1 4.2 11 7.1V3zM5 19h14v2H5v-2z"/></svg>',
    function:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M7 4h7.5a3.5 3.5 0 0 1 0 7H10v2h5v2h-5v5H8v-5H5v-2h3v-2H5V9h3V4zm3 2v3h4.5a1.5 1.5 0 0 0 0-3H10z"/></svg>',
    image:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 5h16a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1zm1 10.5 3.5-3.5 2.5 2.5 4-4L19 15.5V7H5v8.5zM8.5 10a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z"/></svg>',
    sparkle:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2l1.2 5.2L18 8l-4.8 1.2L12 14l-1.2-4.8L6 8l4.8-.8L12 2zm6 8 1 3 3 1-3 1-1 3-1-3-3-1 3-1 1-3zM6 14l.8 2.4L9 17l-2.2.6L6 20l-.8-2.4L3 17l2.2-.6L6 14z"/></svg>',
    board:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 4h5v16H4V4zm6 0h5v10h-5V4zm6 0h4v7h-4V4z"/></svg>',
    help:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a10 10 0 1 1 0 20 10 10 0 0 1 0-20zm0 15a1.25 1.25 0 1 0 0 2.5A1.25 1.25 0 0 0 12 17zm0-11a4 4 0 0 0-4 4v2h1.5a2.5 2.5 0 1 1 3.4 2.3c-.9.4-1.9 1.2-1.9 2.7V15h1.7v-.8c0-.7.5-1.1 1.3-1.5A4 4 0 0 0 12 6z"/></svg>',
    brand:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 6.5A2.5 2.5 0 0 1 6.5 4H12l8 8-5.5 5.5L4 12.5V6.5zm3 .5v4.2l3.8 2.5L17.2 8H7.5A.5.5 0 0 0 7 7z"/></svg>',
    /* Product marks — distinct silhouettes + accent via CSS currentColor / .era-mod-* */
    navDrive:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 3 2.5 9.2 12 21l9.5-11.8L12 3zm0 2.4 5.7 3.7L12 17.6 6.3 9.1 12 5.4z"/><path fill="currentColor" opacity=".45" d="M12 9.2 7.8 12.2 12 17.2l4.2-5z"/></svg>',
    navDocs:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M6 2.5h8.2L20 8.3V21.5H6V2.5zm8 1.7v4.6h4.6L14 4.2z"/><path fill="currentColor" opacity=".55" d="M8.5 11h9v1.4h-9V11zm0 3h9v1.4h-9V14zm0 3h6v1.4h-6V17z"/></svg>',
    navTables:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 4h18v16H3V4zm1.5 1.5v3h5v-3h-5zm6.5 0v3h8.5v-3H11zm-6.5 4.5v4h5v-4h-5zm6.5 0v4h8.5v-4H11zm-6.5 5.5v3.5h5V15h-5zm6.5 0v3.5h8.5V15H11z"/></svg>',
    navPres:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M2.5 4h19v12.5h-19V4zm1.7 1.6v9.3h15.6V5.6H4.2z"/><path fill="currentColor" d="M11 17.2h2v2.2h5.2V21H5.8v-1.6H11v-2.2z"/><path fill="currentColor" opacity=".5" d="M8 8.2h8v1.5H8V8.2zm0 3h5.5v1.5H8V11.2z"/></svg>',
    navProjects:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 5h4.5v14H4V5zm6 3h4.5v11H10V8zm6-1.5H20V19h-4V6.5z"/><path fill="currentColor" opacity=".4" d="M5.2 7.2h2.1v3.2H5.2V7.2zm6 4.3h2.1v4H11.2v-4zm6-2.2H19v6.5h-1.8V9.3z"/></svg>',
    navAI:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2.2l1.6 5.8L19.5 9l-5.9 1.7L12 16.6l-1.6-5.9L4.5 9l5.9-1L12 2.2z"/><path fill="currentColor" opacity=".55" d="M18.2 13.5l.8 2.6 2.6.8-2.6.8-.8 2.6-.8-2.6-2.6-.8 2.6-.8.8-2.6z"/></svg>',
    navMail:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5.5h18V18.5H3V5.5zm1.7 1.6v.5l7.3 4.4 7.3-4.4v-.5H4.7zm0 2.7V17h14.6V9.8l-7.3 4.4-7.3-4.4z"/></svg>',
    more:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M6 10a2 2 0 1 1 0 4 2 2 0 0 1 0-4zm6 0a2 2 0 1 1 0 4 2 2 0 0 1 0-4zm6 0a2 2 0 1 1 0 4 2 2 0 0 1 0-4z"/></svg>',
    open:
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M14 3h7v7h-2V6.4l-9.3 9.3-1.4-1.4L17.6 5H14V3zM5 5h7v2H7v10h10v-5h2v7H5V5z"/></svg>',
  };

  var NAV_BY_MOD = {
    drive: 'navDrive',
    docs: 'navDocs',
    tables: 'navTables',
    pres: 'navPres',
    projects: 'navProjects',
    ai: 'navAI',
    mail: 'navMail',
  };

  var CMD_ICONS = {
    'file.new': 'add',
    'file.open': 'folder',
    'file.openDrive': 'folder',
    'file.share': 'share',
    'file.import': 'upload',
    'file.export': 'download',
    'file.save': 'save',
    'file.refresh': 'refresh',
    'file.rename': 'rename',
    'file.clear': 'trash',
    'file.pageSetup': 'settings',
    'file.pdf': 'file',
    'file.odt': 'file',
    'file.ods': 'table',
    'file.odp': 'present',
    'file.rtf': 'file',
    'file.csv': 'table',
    'file.print': 'print',
    'file.versions': 'history',
    'edit.undo': 'undo',
    'edit.redo': 'redo',
    'edit.cut': 'cut',
    'edit.copy': 'copy',
    'edit.paste': 'paste',
    'edit.pastePlain': 'paste',
    'edit.find': 'search',
    'edit.replace': 'search',
    'edit.selectAll': 'file',
    'edit.fillDown': 'move',
    'edit.fillRight': 'move',
    'edit.clear': 'trash',
    'edit.delRow': 'trash',
    'edit.delCol': 'trash',
    'edit.add': 'add',
    'edit.filter': 'filter',
    'edit.labels': 'list',
    'edit.checklist': 'list',
    'view.printLayout': 'file',
    'view.wordCount': 'file',
    'view.freeze': 'freeze',
    'view.formula': 'function',
    'view.freezePanes': 'freeze',
    'view.present': 'present',
    'view.filmstrip': 'present',
    'view.board': 'board',
    'view.gantt': 'chart',
    'view.swimlanes': 'board',
    'insert.link': 'link',
    'insert.comment': 'comment',
    'insert.pageBreak': 'file',
    'insert.header': 'file',
    'insert.footer': 'file',
    'insert.pageNumbers': 'numbered',
    'insert.image': 'image',
    'insert.table': 'table',
    'insert.toc': 'list',
    'insert.bookmark': 'link',
    'insert.section': 'file',
    'insert.footnote': 'comment',
    'insert.textbox': 'file',
    'insert.sum': 'function',
    'insert.count': 'function',
    'insert.countif': 'function',
    'insert.if': 'function',
    'insert.row': 'add',
    'insert.col': 'add',
    'insert.chart': 'chart',
    'insert.sparkline': 'chart',
    'format.bold': 'bold',
    'format.italic': 'italic',
    'format.underline': 'underline',
    'format.strike': 'strike',
    'format.font': 'file',
    'format.size': 'file',
    'format.title': 'file',
    'format.h1': 'list',
    'format.h2': 'list',
    'format.h3': 'list',
    'format.h4': 'list',
    'format.h5': 'list',
    'format.h6': 'list',
    'format.textMenu': 'bold',
    'format.stylesMenu': 'list',
    'format.alignMenu': 'alignLeft',
    'file.downloadMenu': 'download',
    'format.alignLeft': 'alignLeft',
    'format.alignCenter': 'alignCenter',
    'format.alignRight': 'alignRight',
    'format.alignJustify': 'alignLeft',
    'format.bullet': 'list',
    'format.numbered': 'numbered',
    'format.clear': 'trash',
    'format.number': 'function',
    'format.percent': 'function',
    'format.plain': 'file',
    'format.merge': 'table',
    'format.color': 'sparkle',
    'format.highlight': 'sparkle',
    'format.styles': 'settings',
    'format.language': 'settings',
    'format.columns': 'table',
    'insert.fnMenu': 'function',
    'insert.sheetMenu': 'table',
    'data.sort': 'sort',
    'data.filter': 'filter',
    'data.filterOpts': 'filter',
    'data.subtotal': 'function',
    'data.protect': 'settings',
    'data.protectRanges': 'settings',
    'data.protectMenu': 'settings',
    'data.analysisMenu': 'chart',
    'view.suggest': 'comment',
    'view.fullscreen': 'present',
    'view.lineNumbers': 'numbered',
    'data.whatif': 'chart',
    'data.scenarios': 'settings',
    'data.consolidate': 'function',
    'slide.new': 'add',
    'slide.up': 'move',
    'slide.down': 'move',
    'slide.delete': 'trash',
    'slide.bg': 'image',
    'slide.master': 'settings',
    'tools.wordCount': 'file',
    'tools.summarize': 'sparkle',
    'tools.rewrite': 'sparkle',
    'tools.spelling': 'spell',
    'tools.review': 'comment',
    'tools.compare': 'copy',
    'tools.merge': 'table',
    'help.shortcuts': 'help',
    'help.about': 'help',
  };

  function svgFor(name) {
    var raw = ICONS[name];
    if (!raw) return '';
    return raw.replace('<svg', '<svg class="era-icon"');
  }

  function mount(root) {
    var scope = root && root.querySelectorAll ? root : document;
    scope.querySelectorAll('[data-icon]').forEach(function (el) {
      if (el.querySelector('svg.era-icon')) return;
      var name = el.getAttribute('data-icon');
      var svg = svgFor(name);
      if (!svg) return;
      el.insertAdjacentHTML('afterbegin', svg);
    });
  }

  function mountNav(root) {
    var scope = root && root.querySelectorAll ? root : document;
    scope.querySelectorAll('.era-nav a').forEach(function (a) {
      var mod = a.getAttribute('data-mod');
      if (!mod && /\/mail\/?$/.test(a.getAttribute('href') || '')) mod = 'mail';
      if (!mod) return;
      a.classList.add('era-nav-link');
      if (!a.getAttribute('data-icon') && NAV_BY_MOD[mod]) {
        a.setAttribute('data-icon', NAV_BY_MOD[mod]);
      }
      if (!a.querySelector('.era-nav-label')) {
        var label = a.textContent.trim();
        a.textContent = '';
        var span = document.createElement('span');
        span.className = 'era-nav-label';
        span.textContent = label;
        a.appendChild(span);
      }
    });
    mount(scope);
  }

  function mountMenuIcons(root) {
    var scope = root && root.querySelectorAll ? root : document;
    scope.querySelectorAll('.era-menu-item').forEach(function (item) {
      if (item.querySelector('svg.era-icon') || item.querySelector('.era-icon-slot')) return;
      var cmd = item.getAttribute('data-cmd');
      var icon = item.getAttribute('data-icon') || (cmd && CMD_ICONS[cmd]);
      if (icon) {
        item.setAttribute('data-icon', icon);
        var svg = svgFor(icon);
        if (svg) {
          item.insertAdjacentHTML('afterbegin', svg);
          return;
        }
      }
      // Placeholder keeps label column aligned with iconed items
      item.insertAdjacentHTML('afterbegin', '<span class="era-icon-slot" aria-hidden="true"></span>');
    });
  }

  global.EraOfficeIcons = {
    mount: mount,
    mountNav: mountNav,
    mountMenuIcons: mountMenuIcons,
    icons: ICONS,
    cmdIcons: CMD_ICONS,
    navByMod: NAV_BY_MOD,
  };
})(typeof window !== 'undefined' ? window : globalThis);
