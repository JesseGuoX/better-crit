/* global markdownit, DOMPurify */
(function () {
  'use strict';
  const $ = id => document.getElementById(id);
  const labels = { pending: 'Pending', decided: 'Decided', revision_requested: 'Needs revision' };
  const md = markdownit({ html: false, linkify: true, breaks: false });
  // Decision text is untrusted agent input. Images and raw HTML are unnecessary here.
  md.disable('image');
  let state = null;
  let round = null;
  let draft = {};
  let generation = 0;
  let savedGeneration = 0;
  let timer;
  let saving = null;
  let refreshing = null;
  let submitting = false;
  let conflict = false;
  let remotePending = false;
  let submissionID = null;

  function element(tag, className, text) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined) node.textContent = text;
    return node;
  }
  function markdown(text) {
    const node = element('div', 'markdown');
    node.innerHTML = DOMPurify.sanitize(md.render(text || ''));
    node.querySelectorAll('a').forEach(link => { link.target = '_blank'; link.rel = 'noopener noreferrer'; });
    return node;
  }
  function status(answer) {
    return answer.feedback.trim() ? 'revision_requested' : answer.selected.length ? 'decided' : 'pending';
  }
  async function api(path, method = 'GET', body) {
    const response = await fetch('/api/decision' + path, {
      method, headers: { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body), cache: 'no-store',
    });
    if (!response.ok) {
      const error = new Error(await response.text());
      error.status = response.status;
      throw error;
    }
    return response.json();
  }
  function showError(error) {
    if (error.status === 409) conflict = true;
    $('error-text').textContent = conflict
      ? 'The checklist or draft changed in another tab. Your changes are still on this page. Review them before loading the latest version.'
      : 'Could not save or connect. Your changes are still on this page. Please try again.' + (error.message ? ' ' + error.message : '');
    $('error').hidden = false;
    $('save-status').textContent = conflict ? 'Version conflict · Not submitted' : 'Connection lost · Save not confirmed';
    updateControls();
  }
  function updateControls() {
    const readOnly = !!round?.submission || conflict || submitting || !!submissionID;
    $('items').querySelectorAll('input, textarea, button').forEach(node => { node.disabled = readOnly; });
    $('submit').disabled = !round || !!round.submission || conflict || submitting;
    $('submit').textContent = round?.submission ? 'Submitted' : submitting ? 'Submitting…' : submissionID ? 'Retry submission' : 'Submit decisions';
  }
  function updateCounts() {
    if (!round) return;
    const counts = { pending: 0, decided: 0, revision_requested: 0 };
    round.checklist.items.forEach((item, i) => {
      const current = status(draft[item.id]);
      counts[current]++;
      const badge = $('item-status-' + i);
      if (badge) badge.textContent = current === 'decided' && !round.submission ? 'Selected · Not submitted' : labels[current];
    });
    $('counts').replaceChildren(...['decided', 'revision_requested', 'pending'].map(key => {
      const row = element('div', 'count-row');
      row.append(element('span', '', key === 'decided' && !round.submission ? 'Selected' : labels[key]), element('strong', '', String(counts[key])));
      return row;
    }));
  }
  function changed() {
    generation++;
    updateCounts();
    $('save-status').textContent = 'Saving draft…';
    clearTimeout(timer);
    timer = setTimeout(() => { flush().catch(showError); }, 250);
  }
  async function flush() {
    if (saving) return saving;
    saving = (async () => {
      while (savedGeneration < generation) {
        if (conflict || round.submission) throw new Error('Please load the latest version');
        const sentGeneration = generation;
        const next = await api('/draft', 'PUT', { revision: round.revision, draft_version: round.draft_version, draft: structuredClone(draft) });
        round.draft_version = next.rounds.at(-1).draft_version;
        round.draft = next.rounds.at(-1).draft;
        savedGeneration = sentGeneration;
      }
      if (!round.submission) $('save-status').textContent = 'Draft saved · Submit to send it to the agent';
      $('error').hidden = true;
    })();
    try { await saving; }
    finally {
      saving = null;
      if (remotePending && !submitting && savedGeneration === generation) void refresh();
    }
  }
  function renderItem(item, i) {
    const answer = draft[item.id];
    const card = element('section', 'decision-card');
    card.id = 'item-' + i;
    card.setAttribute('aria-labelledby', 'question-' + i);
    const meta = element('div', 'card-meta');
    meta.append(element('span', 'item-number', String(i + 1).padStart(2, '0')), element('span', '', item.type === 'single' ? 'Single choice' : 'Multiple choice'));
    if (round.changed.includes(item.id)) meta.append(element('span', 'badge changed', 'Updated · Please decide again'));
    if (round.carried.includes(item.id)) meta.append(element('span', 'badge', 'Carried over from the previous round'));
    const badge = element('span', 'item-status'); badge.id = 'item-status-' + i; meta.append(badge);
    const title = element('h2', '', item.title); title.id = 'question-' + i;
    card.append(meta, title);
    if (item.context) card.append(markdown(item.context));
    const fieldset = element('fieldset');
    fieldset.append(element('legend', '', item.type === 'single' ? 'Choose one option, or leave this pending' : 'Choose all that apply, or leave this pending'));
    const options = element('div', 'options');
    item.options.forEach((option, j) => {
      const box = element('div', 'option');
      const label = element('label');
      const input = element('input');
      input.type = item.type === 'single' ? 'radio' : 'checkbox';
      input.name = 'choice-' + i;
      input.value = option.id;
      input.checked = answer.selected.includes(option.id);
      input.addEventListener('change', () => {
        answer.selected = Array.from(options.querySelectorAll('input:checked'), node => node.value);
        changed();
      });
      label.append(input, element('span', '', option.label));
      box.append(label);
      if ((item.recommended || []).includes(option.id)) box.append(element('span', 'badge', 'Agent recommendation'));
      if (option.description) {
        const description = markdown(option.description); description.id = 'description-' + i + '-' + j;
        input.setAttribute('aria-describedby', description.id); box.append(description);
      }
      options.append(box);
    });
    fieldset.append(options);
    card.append(fieldset);
    if (item.recommendation_reason) {
      const recommendation = element('div', 'recommendation');
      recommendation.append(element('strong', '', 'Why this is recommended'), markdown(item.recommendation_reason));
      card.append(recommendation);
    }
    const actions = element('div', 'item-actions');
    const clear = element('button', 'clear', 'Clear selection'); clear.type = 'button';
    clear.addEventListener('click', () => {
      answer.selected = []; options.querySelectorAll('input').forEach(input => { input.checked = false; }); changed();
    });
    actions.append(clear); card.append(actions);
    const feedback = element('details', 'feedback');
    feedback.open = !!answer.feedback;
    feedback.append(element('summary', '', 'Request changes'));
    const label = element('label', '', 'Adding feedback marks this item as needing revision. Selected options only guide the changes.');
    label.htmlFor = 'feedback-' + i;
    const textarea = element('textarea'); textarea.id = label.htmlFor; textarea.value = answer.feedback;
    textarea.maxLength = 25000;
    textarea.setAttribute('aria-label', item.title + ': Revision feedback');
    textarea.placeholder = 'For example: Keep option A, but add offline support…';
    textarea.addEventListener('input', () => { answer.feedback = textarea.value; changed(); });
    feedback.append(label, textarea); card.append(feedback);
    return card;
  }
  function renderHistory() {
    $('history').hidden = state.rounds.length < 2;
    $('history-content').replaceChildren(...state.rounds.slice(0, -1).reverse().map(previous => {
      const section = element('section', 'history-round');
      section.append(element('h3', '', 'Round ' + previous.revision + ' · ' + previous.checklist.title + (previous.submission ? ' · Submitted' : ' · Unsubmitted draft')));
      if (previous.checklist.context) section.append(markdown(previous.checklist.context));
      const list = element('ul');
      previous.checklist.items.forEach(item => {
        const result = previous.submission?.items.find(answer => answer.id === item.id);
        const answer = result || previous.draft[item.id];
        const selected = answer.selected.map(id => item.options.find(option => option.id === id)?.label || id).join(', ');
        const row = element('li', '', item.title + ' · ' + (result ? labels[result.status] : 'Not submitted') + (selected ? '\n' + selected : '') + (answer.feedback ? '\nFeedback: ' + answer.feedback : ''));
        list.append(row);
      });
      section.append(list); return section;
    }));
  }
  function render(next) {
    state = next; round = state.rounds.at(-1);
    if (!round) { $('title').textContent = 'Waiting for a decision checklist from the agent'; return; }
    draft = structuredClone(round.draft);
    generation = 0; savedGeneration = 0; conflict = false; submissionID = null;
    $('error').hidden = true;
    $('title').textContent = round.checklist.title;
    document.title = round.checklist.title + ' · crit+ Decisions';
    $('context').replaceChildren(markdown(round.checklist.context));
    $('revision').textContent = 'Round ' + round.revision;
    $('items').replaceChildren(...round.checklist.items.map(renderItem));
    $('navigation').replaceChildren(...round.checklist.items.map((item, i) => {
      const link = element('a', '', String(i + 1).padStart(2, '0') + '  ' + item.title); link.href = '#item-' + i; return link;
    }));
    $('notice').hidden = !round.submission;
    $('notice').textContent = round.submission?.completed ? 'All items in this round are decided. Results have been saved and sent to the agent.' : 'This round is submitted. Waiting for the agent to update the checklist. Pending items and items needing revision are not approved.';
    $('save-status').textContent = round.submission ? 'Submission saved' : 'Draft saved · Submit to send it to the agent';
    $('submit-hint').textContent = round.submission ? 'This version is locked. The page will update automatically when a new version arrives.' : 'Submit anytime. Unanswered items remain pending.';
    updateCounts(); updateControls(); renderHistory();
  }
  async function refresh() {
    if (refreshing || saving || submitting) { remotePending = true; return; }
    remotePending = false;
    refreshing = (async () => {
      const next = await api('');
      // A user may have edited while this GET was in flight.
      if (saving || submitting) { remotePending = true; return; }
      const current = next.rounds.at(-1);
      const differs = !round || current?.revision !== round.revision || current?.draft_version !== round.draft_version || current?.submission?.submission_id !== round.submission?.submission_id;
      if (!differs) {
        if (generation === savedGeneration && $('error').hidden) {
          $('save-status').textContent = round?.submission ? 'Submission saved' : 'Draft saved · Submit to send it to the agent';
        }
        return;
      }
      if (generation !== savedGeneration || (submissionID && current?.submission?.submission_id !== submissionID)) { showError({ status: 409 }); return; }
      render(next);
    })();
    try { await refreshing; } catch (error) { showError(error); }
    finally {
      refreshing = null;
      if (remotePending && !saving && !submitting) void refresh();
    }
  }
  $('submit').addEventListener('click', async () => {
    if (submitting || conflict || !round || round.submission) return;
    submitting = true; updateControls(); clearTimeout(timer);
    try {
      await flush();
      submissionID ||= crypto.randomUUID();
      const result = await api('/submit', 'POST', { revision: round.revision, draft_version: round.draft_version, submission_id: submissionID });
      round.submission = result;
      render(state);
    } catch (error) { showError(error); }
    finally { submitting = false; updateControls(); void refresh(); }
  });
  $('reload').addEventListener('click', async () => {
    if (saving || submitting) return;
    // Explicitly discard only after the user chooses to load the current version.
    try { render(await api('')); } catch (error) { showError(error); }
  });
  $('theme').value = window.crit.shared.getSetting('theme', 'system');
  $('theme').addEventListener('change', () => {
    window.crit.shared.setSetting('theme', $('theme').value);
    window.crit.shared.applyThemeFromCookie();
  });
  window.addEventListener('beforeunload', event => {
    if (generation !== savedGeneration) { event.preventDefault(); event.returnValue = ''; }
  });
  void refresh();
  const events = new EventSource('/api/decision/events');
  events.addEventListener('decision-updated', () => { void refresh(); });
  events.onerror = () => { if (round && !round.submission && !saving) $('save-status').textContent = 'Reconnecting… Your draft is still on this page'; };
})();
