'use strict';
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { isCritWaitCommand, roundReadyToast } = require('./crit-plus-wait-notify.js');

test('isCritWaitCommand accepts bare crit-plus and file args', () => {
  assert.equal(isCritWaitCommand('crit-plus'), true);
  assert.equal(isCritWaitCommand('crit-plus plan.md'), true);
  assert.equal(isCritWaitCommand('./crit-plus'), true);
  assert.equal(isCritWaitCommand('crit-plus --pr 12'), true);
  assert.equal(isCritWaitCommand('CRIT_FOO=1 crit-plus'), true);
});

test('isCritWaitCommand rejects non-wait subcommands', () => {
  assert.equal(isCritWaitCommand('crit-plus comment --reply-to c_1 hi'), false);
  assert.equal(isCritWaitCommand('crit-plus comments --json'), false);
  assert.equal(isCritWaitCommand('crit-plus config'), false);
  assert.equal(isCritWaitCommand('crit-plus share plan.md'), false);
  assert.equal(isCritWaitCommand('crit-plus install opencode'), false);
  assert.equal(isCritWaitCommand('crit-plus status'), false);
  assert.equal(isCritWaitCommand('crit-plus stop'), false);
  assert.equal(isCritWaitCommand('crit-plus check'), false);
  assert.equal(isCritWaitCommand('crit-plus stats'), false);
  assert.equal(isCritWaitCommand('crit-plus cleanup'), false);
  assert.equal(isCritWaitCommand('crit-plus auth login'), false);
  assert.equal(isCritWaitCommand('crit-plus fetch abc'), false);
  assert.equal(isCritWaitCommand('echo crit-plus'), false);
  assert.equal(isCritWaitCommand(''), false);
});

test('isCritWaitCommand accepts live/preview/review/plan waits', () => {
  assert.equal(isCritWaitCommand('crit-plus live http://127.0.0.1:3000'), true);
  assert.equal(isCritWaitCommand('crit-plus preview ./dist'), true);
  assert.equal(isCritWaitCommand('crit-plus review'), true);
  assert.equal(isCritWaitCommand('crit-plus plan my-plan'), true);
});

test('roundReadyToast includes URL when provided', () => {
  const withURL = roundReadyToast('http://127.0.0.1:9');
  assert.equal(withURL.title, 'Crit Plus');
  assert.match(withURL.message, /http:\/\/127\.0\.0\.1:9/);

  const without = roundReadyToast('');
  assert.match(without.message, /browser/i);
});
