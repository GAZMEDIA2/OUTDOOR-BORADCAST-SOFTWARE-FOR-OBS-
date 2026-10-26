/**
 * Page Setup Studio – a Google Docs add-on that sets any page size,
 * margins and gutter, for the current tab or every tab, with undo.
 *
 * All sizes on the server side are in points (1 inch = 72 pt).
 * Page setup never touches document text. The optional drop-cap fix only
 * replaces big chapter first letters; nothing is ever sent anywhere.
 */

var APP_NAME = 'Page Setup Studio';
var MIN_CONTENT_PT = 36;   // keep at least 0.5 inch of writing space
var MIN_PAGE_PT = 72;      // 1 inch
var MAX_PAGE_PT = 120 * 72; // 120 inches
var UNDO_KEY = 'pss_undo';
var SAVED_KEY = 'pss_saved';
var UNDO_DEPTH = 10;
var DEFAULT_PAGE = { w: 612, h: 792, top: 72, bottom: 72, left: 72, right: 72 }; // US Letter, 1" margins

/* ---------------- menu and sidebar ---------------- */

function onOpen(e) {
  DocumentApp.getUi().createAddonMenu()
    .addItem('Open page setup', 'showSidebar')
    .addToUi();
}

function onInstall(e) {
  onOpen(e);
}

function showSidebar() {
  var html = HtmlService.createHtmlOutputFromFile('Sidebar').setTitle(APP_NAME);
  DocumentApp.getUi().showSidebar(html);
}

/* ---------------- tabs ---------------- */

/** Lists every document tab (including child tabs) as {id, title, depth, body}. */
function listTabs_(doc) {
  var out = [];
  if (typeof doc.getTabs !== 'function') {
    out.push({ id: '', title: 'Document', depth: 0, body: doc.getBody() });
    return out;
  }
  var walk = function (tabs, depth) {
    for (var i = 0; i < tabs.length; i++) {
      var t = tabs[i];
      if (t.getType() === DocumentApp.TabType.DOCUMENT_TAB) {
        out.push({ id: t.getId(), title: t.getTitle(), depth: depth, body: t.asDocumentTab().getBody() });
      }
      walk(t.getChildTabs(), depth + 1);
    }
  };
  walk(doc.getTabs(), 0);
  return out;
}

function activeTabId_(doc) {
  try {
    var t = doc.getActiveTab();
    return t ? t.getId() : '';
  } catch (err) {
    return '';
  }
}

function findTab_(tabs, id) {
  for (var i = 0; i < tabs.length; i++) if (tabs[i].id === id) return tabs[i];
  return tabs[0];
}

/* ---------------- reading and writing page setup ---------------- */

function num_(v, fallback) {
  return (typeof v === 'number' && isFinite(v) && v > 0) ? v : fallback;
}

function readPage_(body) {
  return {
    w: num_(body.getPageWidth(), DEFAULT_PAGE.w),
    h: num_(body.getPageHeight(), DEFAULT_PAGE.h),
    top: num_(body.getMarginTop(), 0),
    bottom: num_(body.getMarginBottom(), 0),
    left: num_(body.getMarginLeft(), 0),
    right: num_(body.getMarginRight(), 0)
  };
}

function setSize_(body, p) {
  body.setPageWidth(p.w);
  body.setPageHeight(p.h);
}

function setMargins_(body, p) {
  body.setMarginTop(p.top);
  body.setMarginBottom(p.bottom);
  body.setMarginLeft(p.left);
  body.setMarginRight(p.right);
}

/**
 * Writes page setup. Docs can refuse a size that is smaller than the
 * current margins (or margins larger than the current size), so the order
 * is chosen to keep every intermediate step valid.
 */
function writePage_(body, p) {
  var cur = readPage_(body);
  var shrinking = p.w < cur.w || p.h < cur.h;
  if (shrinking) {
    setMargins_(body, p);
    setSize_(body, p);
  } else {
    setSize_(body, p);
    setMargins_(body, p);
  }
}

/* ---------------- validation (pure, unit-tested) ---------------- */

/**
 * Checks a request in points. Returns {ok:true, page} or {ok:false, error}.
 * The gutter is added to the left (inside) margin, as Google Docs has no
 * separate gutter setting.
 */
function validatePage(req) {
  var keys = ['w', 'h', 'top', 'bottom', 'left', 'right'];
  var p = {};
  for (var i = 0; i < keys.length; i++) {
    var v = Number(req[keys[i]]);
    if (!isFinite(v)) return { ok: false, error: 'Please enter a number for every size and margin.' };
    p[keys[i]] = v;
  }
  var gutter = Number(req.gutter || 0);
  if (!isFinite(gutter) || gutter < 0) return { ok: false, error: 'The gutter cannot be negative.' };
  if (p.w < MIN_PAGE_PT || p.h < MIN_PAGE_PT) return { ok: false, error: 'The page must be at least 1 inch (25.4 mm) on each side.' };
  if (p.w > MAX_PAGE_PT || p.h > MAX_PAGE_PT) return { ok: false, error: 'The page can be at most 120 inches (3048 mm) on each side.' };
  if (p.top < 0 || p.bottom < 0 || p.left < 0 || p.right < 0) return { ok: false, error: 'Margins cannot be negative.' };
  p.left = p.left + gutter;
  if (p.w - p.left - p.right < MIN_CONTENT_PT) return { ok: false, error: 'The left and right margins (with the gutter) leave less than 0.5 inch to write in. Make them smaller.' };
  if (p.h - p.top - p.bottom < MIN_CONTENT_PT) return { ok: false, error: 'The top and bottom margins leave less than 0.5 inch to write in. Make them smaller.' };
  for (var k in p) p[k] = Math.round(p[k] * 100) / 100;
  return { ok: true, page: p };
}

/* ---------------- functions called by the sidebar ---------------- */

function getState() {
  var doc = DocumentApp.getActiveDocument();
  var tabs = listTabs_(doc);
  var activeId = activeTabId_(doc);
  var active = findTab_(tabs, activeId);
  return {
    tabs: tabs.map(function (t) { return { id: t.id, title: t.title, depth: t.depth }; }),
    activeTabId: active.id,
    activeTitle: active.title,
    page: readPage_(active.body),
    saved: getSavedSizes_(),
    canUndo: getUndoStack_().length > 0,
    canUndoDc: loadDcStack_().length > 0
  };
}

/** req: {w,h,top,bottom,left,right,gutter} in points, plus allTabs (bool). */
function applyPage(req) {
  var check = validatePage(req);
  if (!check.ok) throw new Error(check.error);
  var doc = DocumentApp.getActiveDocument();
  var tabs = listTabs_(doc);
  var targets = req.allTabs ? tabs : [findTab_(tabs, activeTabId_(doc))];

  var before = targets.map(function (t) { return { id: t.id, page: readPage_(t.body) }; });
  var done = [];
  try {
    for (var i = 0; i < targets.length; i++) {
      writePage_(targets[i].body, check.page);
      done.push(before[i]);
    }
  } catch (err) {
    // put back whatever was already changed, so the document is never half-done
    restore_(tabs, done);
    throw new Error('Google Docs did not accept this page setup (' + err.message + '). Nothing was changed.');
  }
  pushUndo_(before);
  var st = getState();
  st.message = 'Done: ' + (targets.length > 1 ? targets.length + ' tabs' : 'this tab') + ' changed.';
  return st;
}

function undoLast() {
  var stack = getUndoStack_();
  if (!stack.length) throw new Error('There is nothing to undo.');
  var last = stack.pop();
  var tabs = listTabs_(DocumentApp.getActiveDocument());
  restore_(tabs, last);
  saveUndoStack_(stack);
  var st = getState();
  st.message = 'Undone: the previous page setup is back.';
  return st;
}

function restore_(tabs, entries) {
  for (var i = 0; i < entries.length; i++) {
    for (var j = 0; j < tabs.length; j++) {
      if (tabs[j].id === entries[i].id) writePage_(tabs[j].body, entries[i].page);
    }
  }
}

/* ---------------- undo history (per document) ---------------- */

function getUndoStack_() {
  var raw = PropertiesService.getDocumentProperties().getProperty(UNDO_KEY);
  try { return raw ? JSON.parse(raw) : []; } catch (e) { return []; }
}

function pushUndo_(entry) {
  var stack = getUndoStack_();
  stack.push(entry);
  while (stack.length > UNDO_DEPTH) stack.shift();
  saveUndoStack_(stack);
}

function saveUndoStack_(stack) {
  var props = PropertiesService.getDocumentProperties();
  var json = JSON.stringify(stack);
  // a property holds at most 9 KB; drop the oldest steps if needed
  while (json.length > 8500 && stack.length > 1) { stack.shift(); json = JSON.stringify(stack); }
  props.setProperty(UNDO_KEY, json);
}

/* ---------------- the user's own saved sizes (follow the user) ---------------- */

function getSavedSizes_() {
  var raw = PropertiesService.getUserProperties().getProperty(SAVED_KEY);
  try { return raw ? JSON.parse(raw) : []; } catch (e) { return []; }
}

/** size: {name, w, h, top, bottom, left, right, gutter, unit} with lengths in points. */
function saveSize(size) {
  var name = String(size.name || '').trim().slice(0, 40);
  if (!name) throw new Error('Give the size a name first.');
  var list = getSavedSizes_().filter(function (s) { return s.name !== name; });
  list.push({
    name: name, unit: size.unit || 'in',
    w: +size.w, h: +size.h, top: +size.top, bottom: +size.bottom,
    left: +size.left, right: +size.right, gutter: +(size.gutter || 0)
  });
  list.sort(function (a, b) { return a.name.localeCompare(b.name); });
  if (list.length > 50) throw new Error('You can keep up to 50 sizes. Delete one first.');
  PropertiesService.getUserProperties().setProperty(SAVED_KEY, JSON.stringify(list));
  return list;
}

function deleteSize(name) {
  var list = getSavedSizes_().filter(function (s) { return s.name !== name; });
  PropertiesService.getUserProperties().setProperty(SAVED_KEY, JSON.stringify(list));
  return list;
}

/* ================= drop caps ================= */
/*
 * Google Docs has no drop caps. When a book made in Word or InDesign is
 * uploaded, each chapter's big first letter becomes a paragraph of its own
 * (huge font, huge line spacing) and the chapter text is pushed down.
 * This finds those letters and replaces each one with a picture of the same
 * letter, placed inside the paragraph with the text wrapping around it.
 */
var DC_UNDO_KEY = 'pss_dc_undo';
var DC_MIN_RATIO = 1.8;      // letter must be at least this much bigger than the text after it

function isParagraph_(el) {
  var t = el.getType();
  return t === DocumentApp.ElementType.PARAGRAPH || t === DocumentApp.ElementType.LIST_ITEM;
}

function firstCharIndex_(s) {
  for (var i = 0; i < s.length; i++) if (s.charAt(i).trim()) return i;
  return -1;
}

/** Style of the text paragraph a drop cap sits in (what the letter must line up with). */
function bodyStyle_(p) {
  var tx = p.editAsText(), s = tx.getText(), i = Math.max(0, firstCharIndex_(s));
  return {
    size: tx.getFontSize(i) || 11,
    font: tx.getFontFamily(i) || 'Arial',
    line: p.getLineSpacing() || 1.15
  };
}

/** Looks for broken drop caps in one body. */
function scanBody_(body, tabId, tabTitle) {
  var out = [], n = body.getNumChildren();
  for (var i = 0; i < n - 1; i++) {
    var el = body.getChild(i);
    if (!isParagraph_(el)) continue;
    var raw = el.asText().getText(), s = raw.trim();
    if (s.length < 1 || s.length > 3 || !/[A-Za-z0-9À-ɏ]/.test(s)) continue;
    var tx = el.editAsText(), k = firstCharIndex_(raw), size = tx.getFontSize(k);
    if (!size) continue;
    var next = body.getChild(i + 1);
    if (!isParagraph_(next)) continue;
    var ntext = next.asText().getText();
    if (!ntext.trim()) continue;
    var bs = bodyStyle_(next);
    if (size < bs.size * DC_MIN_RATIO) continue;
    out.push({
      tabId: tabId, tab: tabTitle, index: i, letter: s,
      font: tx.getFontFamily(k) || bs.font, bold: !!tx.isBold(k), italic: !!tx.isItalic(k),
      color: tx.getForegroundColor(k) || '#000000',
      bodySize: bs.size, bodyLine: bs.line, bodyFont: bs.font,
      next: ntext.trim().slice(0, 40)
    });
  }
  return out;
}

function findDropCaps(allTabs) {
  var doc = DocumentApp.getActiveDocument(), tabs = listTabs_(doc);
  var targets = allTabs ? tabs : [findTab_(tabs, activeTabId_(doc))];
  var items = [];
  targets.forEach(function (t) { items = items.concat(scanBody_(t.body, t.id, t.title)); });
  return { items: items, tabs: targets.length };
}

/** Where to put the letter: line height of the text, and the drop cap height for N lines (points). */
function dropCapGeometry(bodySize, bodyLine, lines) {
  var lh = bodySize * 1.25 * (bodyLine || 1.15);   // height of one line of text
  var cap = bodySize * 0.72;                        // height of a capital letter in the text
  return {
    lineHeight: Math.round(lh * 100) / 100,
    height: Math.round(((lines - 1) * lh + cap) * 100) / 100,   // top of line 1 capitals to baseline of line N
    top: Math.round((lh - cap) * 0.62 * 100) / 100              // gap above the capitals in line 1
  };
}

function imageBlob_(b64) {
  return Utilities.newBlob(Utilities.base64Decode(b64), 'image/png', 'drop-cap.png');
}

/** Puts the letter picture at the start of paragraph p. img: {b64, w, h, top} in points. */
function placeCap_(p, img) {
  var pos = p.addPositionedImage(imageBlob_(img.b64));
  var pxPerPt = 96 / 72;
  pos.setWidth(Math.round(img.w * pxPerPt)).setHeight(Math.round(img.h * pxPerPt))
     .setLayout(DocumentApp.PositionedLayout.WRAP_TEXT)
     .setLeftOffset(0).setTopOffset(img.top);
  return pos.getId();
}

function cleanAttrs_(a) {
  var o = {};
  for (var k in a) if (a[k] !== null && a[k] !== undefined) o[k] = a[k];
  return o;
}

/**
 * req: {items:[{tabId,index,letter,key}], images:{key:{b64,w,h,top}}}
 * Works from the bottom of each tab up, so earlier positions stay valid.
 */
function fixDropCaps(req) {
  var doc = DocumentApp.getActiveDocument(), tabs = listTabs_(doc);
  var byTab = {};
  req.items.forEach(function (it) { (byTab[it.tabId] = byTab[it.tabId] || []).push(it); });
  var undo = { styles: [], entries: [] }, styleKey = {}, fixed = 0, skipped = 0;
  Object.keys(byTab).forEach(function (tabId) {
    var tab = findTab_(tabs, tabId), body = tab.body;
    byTab[tabId].sort(function (a, b) { return b.index - a.index; }).forEach(function (it) {
      var el = body.getChild(it.index);
      if (!el || !isParagraph_(el) || el.asText().getText().trim() !== it.letter || it.index + 1 >= body.getNumChildren()) { skipped++; return; }
      var next = body.getChild(it.index + 1);
      if (!isParagraph_(next)) { skipped++; return; }
      var raw = el.asText().getText(), k = firstCharIndex_(raw);
      var style = { p: cleanAttrs_(el.getAttributes()), t: cleanAttrs_(el.editAsText().getAttributes(k)), raw: raw };
      var sk = JSON.stringify(style);
      if (!(sk in styleKey)) { styleKey[sk] = undo.styles.length; undo.styles.push(style); }
      var imgId = placeCap_(next, req.images[it.key]);
      el.removeFromParent();
      undo.entries.push({ t: tabId, i: it.index, s: styleKey[sk], g: imgId });
      fixed++;
    });
  });
  if (fixed) saveDcUndo_(undo);
  return { fixed: fixed, skipped: skipped };
}

/** Info about the paragraph holding the cursor, for "make a drop cap here". */
function cursorParagraph() {
  var doc = DocumentApp.getActiveDocument(), c = doc.getCursor() || (doc.getSelection() && doc.getSelection().getRangeElements()[0]);
  if (!c) throw new Error('Click inside the paragraph that should get a drop cap, then try again.');
  var el = c.getElement ? c.getElement() : c;
  while (el && !isParagraph_(el)) el = el.getParent();
  if (!el) throw new Error('Click inside a normal text paragraph, then try again.');
  var raw = el.asText().getText(), k = firstCharIndex_(raw);
  if (k < 0) throw new Error('That paragraph is empty.');
  var tx = el.editAsText(), bs = bodyStyle_(el);
  return { letter: raw.charAt(k), font: tx.getFontFamily(k) || bs.font, bold: !!tx.isBold(k), italic: !!tx.isItalic(k),
           color: tx.getForegroundColor(k) || '#000000', bodySize: bs.size, bodyLine: bs.line, bodyFont: bs.font,
           next: raw.slice(k, k + 40) };
}

function makeDropCapHere(img) {
  var doc = DocumentApp.getActiveDocument(), c = doc.getCursor() || (doc.getSelection() && doc.getSelection().getRangeElements()[0]);
  if (!c) throw new Error('Click inside the paragraph again, then press the button.');
  var el = c.getElement ? c.getElement() : c;
  while (el && !isParagraph_(el)) el = el.getParent();
  var tx = el.editAsText(), raw = tx.getText(), k = firstCharIndex_(raw);
  var style = { t: cleanAttrs_(tx.getAttributes(k)), raw: raw.charAt(k), inline: true, at: k };
  var body = el.getParent(), idx = body.getChildIndex(el);
  var imgId = placeCap_(el, img);
  tx.deleteText(k, k);
  var tabs = listTabs_(doc), tabId = activeTabId_(doc);
  saveDcUndo_({ styles: [style], entries: [{ t: tabId, i: idx, s: 0, g: imgId, inline: true }] });
  return { fixed: 1 };
}

function undoDropCaps() {
  var stack = loadDcStack_(), undo = stack.pop();
  if (!undo || !undo.entries.length) throw new Error('There are no drop cap changes to undo.');
  var tabs = listTabs_(DocumentApp.getActiveDocument()), done = 0;
  // entries were made bottom-up, so put them back top-down
  undo.entries.slice().reverse().forEach(function (e) {
    var body = findTab_(tabs, e.t).body, st = undo.styles[e.s];
    if (e.inline) {
      var p = body.getChild(e.i);
      try { p.removePositionedImage(e.g); } catch (x) {}
      p.editAsText().insertText(st.at, st.raw).setAttributes(st.at, st.at, st.t);
    } else {
      var p2 = body.getChild(e.i);
      try { p2.removePositionedImage(e.g); } catch (x) {}
      var np = body.insertParagraph(e.i, st.raw);
      np.setAttributes(st.p);
      np.editAsText().setAttributes(st.t);
    }
    done++;
  });
  writeDcStack_(stack);
  return { undone: done, more: stack.length > 0 };
}

/* undo history for drop caps: a stack of batches, split over several properties when large */
var DC_UNDO_DEPTH = 10;
function saveDcUndo_(batch) {
  var stack = loadDcStack_();
  stack.push(batch);
  while (stack.length > DC_UNDO_DEPTH) stack.shift();
  writeDcStack_(stack);
}
function writeDcStack_(stack) {
  var props = PropertiesService.getDocumentProperties(), json = JSON.stringify(stack), parts = [];
  for (var i = 0; i < json.length; i += 8000) parts.push(json.slice(i, i + 8000));
  clearDcUndo_();
  var map = {};
  map[DC_UNDO_KEY + '_n'] = String(parts.length);
  parts.forEach(function (p, i) { map[DC_UNDO_KEY + '_' + i] = p; });
  props.setProperties(map);
}
function loadDcStack_() {
  var props = PropertiesService.getDocumentProperties(), n = +(props.getProperty(DC_UNDO_KEY + '_n') || 0), s = '';
  for (var i = 0; i < n; i++) s += props.getProperty(DC_UNDO_KEY + '_' + i) || '';
  try { var v = s ? JSON.parse(s) : []; return Array.isArray(v) ? v : (v && v.entries ? [v] : []); } catch (e) { return []; }
}
function loadDcUndo_() { // the most recent batch, or null
  var stack = loadDcStack_();
  return stack.length ? stack[stack.length - 1] : null;
}
function clearDcUndo_() {
  var props = PropertiesService.getDocumentProperties(), n = +(props.getProperty(DC_UNDO_KEY + '_n') || 0);
  for (var i = 0; i < n; i++) props.deleteProperty(DC_UNDO_KEY + '_' + i);
  props.deleteProperty(DC_UNDO_KEY + '_n');
}

/* so the file can be unit-tested in Node */
if (typeof module !== 'undefined') module.exports = { dropCapGeometry: dropCapGeometry, scanBody_: scanBody_, fixDropCaps: fixDropCaps, undoDropCaps: undoDropCaps, validatePage: validatePage, writePage_: writePage_, readPage_: readPage_ };
