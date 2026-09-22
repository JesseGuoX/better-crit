import { test as base, expect, type APIRequestContext } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { spawn, execFileSync, type ChildProcess } from 'node:child_process';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const binary = process.env.CRIT_BIN || resolve(__dirname, '../../..', process.platform === 'win32' ? 'crit-plus.exe' : 'crit-plus');
const checklist = () => ({
  id: 'release', title: '首版方案裁定', context: '请确认首版范围。**推荐仅供参考**，可以随时提交。',
  items: [
    { id: 'store', title: '数据应该如何保存？', type: 'single', context: '需要支持离线使用。',
      options: [{ id: 'sqlite', label: 'SQLite', description: '单文件，部署简单。' }, { id: 'postgres', label: 'PostgreSQL', description: '支持更多并发，需要服务。' }],
      recommended: ['sqlite'], recommendation_reason: '符合当前单机使用场景。' },
    { id: 'export', title: '需要哪些导出格式？', type: 'multiple',
      options: [{ id: 'json', label: 'JSON', description: '供其他程序读取。' }, { id: 'csv', label: 'CSV', description: '方便表格编辑。' }], recommended: ['json'] },
    { id: 'sync', title: '是否需要同步？', type: 'single', options: [{ id: 'yes', label: '需要' }, { id: 'later', label: '以后再做' }] },
  ],
});
type Checklist = ReturnType<typeof checklist>;
type Result = { completed: boolean; submission_id: string; revision: number; items: { id: string; status: string; selected: string[]; feedback: string }[] };
type Run = { url: string; result: Promise<Result>; process: ChildProcess };
type DecisionFixture = { initial: Run; start: (input: Checklist, file?: boolean, extra?: string[]) => Promise<Run>; stop: () => void };

// This mode has its own durable store and no comment endpoints. Isolate each
// test in a fresh HOME/project instead of clearing a shared review session.
const test = base.extend<{ decisions: DecisionFixture }>({
  decisions: async ({}, use) => {
    const dir = mkdtempSync(join(tmpdir(), 'crit-decision-e2e-'));
    const home = join(dir, 'home'); mkdirSync(home);
    const env = { ...process.env, HOME: home, USERPROFILE: home, CRIT_NO_UPDATE_CHECK: '1', CRIT_NO_INTEGRATION_CHECK: '1' };
    const processes: ChildProcess[] = [];
    const stop = () => { execFileSync(binary, ['stop', '--all'], { cwd: dir, env, timeout: 10_000, stdio: 'pipe' }); };
    const start = async (input: Checklist, file = false, extra: string[] = []): Promise<Run> => {
      const path = join(dir, 'decisions.json');
      if (file) writeFileSync(path, JSON.stringify(input));
      const child = spawn(binary, ['decide', '--no-open', ...extra, file ? path : '-'], { cwd: dir, env, stdio: ['pipe', 'pipe', 'pipe'] });
      processes.push(child);
      let stdout = ''; let stderr = '';
      child.stdout!.on('data', data => { stdout += data.toString(); });
      child.stderr!.on('data', data => { stderr += data.toString(); });
      const result = new Promise<Result>((resolveResult, reject) => {
        child.on('error', reject);
        child.on('close', code => {
          if (code !== 0) { reject(new Error(`crit exited ${code}: ${stderr}`)); return; }
          try { resolveResult(JSON.parse(stdout)); } catch (error) { reject(error); }
        });
      });
      // Some tests deliberately stop a waiting client; handle those rejections.
      void result.catch(() => {});
      child.stdin!.end(file ? undefined : JSON.stringify(input));
      await expect.poll(() => stderr.match(/(http:\/\/[^\s]+)\/decide/)?.[1] || (child.exitCode !== null ? stderr : ''), { timeout: 15_000 }).toMatch(/^http:\/\//);
      return { url: stderr.match(/(http:\/\/[^\s]+)\/decide/)![1], result, process: child };
    };
    try { await use({ initial: await start(checklist()), start, stop }); }
    finally {
      try { stop(); } finally { processes.forEach(child => { if (child.exitCode === null) child.kill(); }); rmSync(dir, { recursive: true, force: true }); }
    }
  },
});
async function state(request: APIRequestContext, url: string) {
  const response = await request.get(url + '/api/decision'); expect(response.ok()).toBeTruthy(); return response.json();
}

test('stdin CLI returns partial snapshot after flushing choices and revision feedback', async ({ page, decisions }) => {
  await page.goto(decisions.initial.url + '/decide');
  await expect(page.locator('.decision-card')).toHaveCount(3);
  await expect(page.locator('input:checked')).toHaveCount(0);
  await page.getByRole('radio', { name: 'SQLite', exact: true }).check();
  await page.getByRole('checkbox', { name: 'JSON', exact: true }).check();
  await page.getByRole('checkbox', { name: 'CSV', exact: true }).check();
  await page.locator('#item-1 summary').click();
  await page.locator('#feedback-1').fill('增加 XML 导出');
  await page.getByRole('button', { name: '提交裁定', exact: true }).click();
  const result = await decisions.initial.result;
  expect(result.completed).toBe(false);
  expect(result.items.map(item => item.status)).toEqual(['decided', 'revision_requested', 'pending']);
  expect(result.items[1].selected).toEqual(['json', 'csv']);
  expect(result.items[1].feedback).toBe('增加 XML 导出');
  await expect(page.getByRole('button', { name: '已提交', exact: true })).toBeDisabled();
  await expect(page.getByRole('radio', { name: 'SQLite', exact: true })).toBeChecked();
  await expect(page.locator('#feedback-1')).toHaveValue('增加 XML 导出');
  await expect(page.locator('#feedback-1')).toBeDisabled();
  const reconnect = await decisions.start(checklist(), true);
  expect(await reconnect.result).toEqual(result);
});

test('all pending is a valid submission and can be explicitly reopened', async ({ page, decisions }) => {
  await page.goto(decisions.initial.url);
  await page.getByRole('button', { name: '提交裁定', exact: true }).click();
  const pending = await decisions.initial.result;
  expect(pending.completed).toBe(false);
  expect(pending.items.every(item => item.status === 'pending')).toBe(true);
  const next = await decisions.start(checklist(), true, ['--new-round']);
  await expect(page.locator('#revision')).toHaveText('第 2 轮');
  await page.getByRole('radio', { name: 'SQLite', exact: true }).check();
  await page.getByRole('checkbox', { name: 'JSON', exact: true }).check();
  await page.getByRole('radio', { name: '以后再做', exact: true }).check();
  await page.getByRole('button', { name: '提交裁定', exact: true }).click();
  expect((await next.result).completed).toBe(true);
  await expect(page.locator('#notice')).toContainText('所有条目已裁定');
});

test('saved drafts survive page reload and daemon restart without releasing the agent', async ({ page, request, decisions }) => {
  await page.goto(decisions.initial.url);
  await page.getByRole('radio', { name: 'PostgreSQL', exact: true }).check();
  await page.locator('#item-0 summary').click();
  await page.locator('#feedback-0').fill('请说明备份方案');
  await expect.poll(async () => (await state(request, decisions.initial.url)).rounds[0].draft.store.feedback).toBe('请说明备份方案');
  expect(decisions.initial.process.exitCode).toBeNull();
  await page.reload();
  await expect(page.locator('#feedback-0')).toHaveValue('请说明备份方案');
  await expect(page.getByRole('radio', { name: 'PostgreSQL', exact: true })).toBeChecked();
  decisions.stop();
  const recovered = await decisions.start(checklist());
  await page.goto(recovered.url);
  await expect(page.locator('#feedback-0')).toHaveValue('请说明备份方案');
  await page.getByRole('button', { name: '提交裁定', exact: true }).click();
  expect((await recovered.result).items[0].status).toBe('revision_requested');
});

test('new rounds carry confirmed choices, invalidate changed options and preserve history', async ({ page, request, decisions }) => {
  await page.goto(decisions.initial.url);
  await page.getByRole('radio', { name: 'SQLite', exact: true }).check();
  await page.getByRole('checkbox', { name: 'JSON', exact: true }).check();
  await page.getByRole('button', { name: '提交裁定', exact: true }).click();
  await decisions.initial.result;
  const changed = checklist(); changed.items[1].options[0].description = 'JSON 现在还包含私人数据。';
  const next = await decisions.start(changed);
  await expect(page.locator('#revision')).toHaveText('第 2 轮');
  await expect(page.getByRole('radio', { name: 'SQLite', exact: true })).toBeChecked();
  await expect(page.getByRole('checkbox', { name: 'JSON', exact: true })).not.toBeChecked();
  await expect(page.locator('#item-1 .changed')).toBeVisible();
  const stale = await request.post(next.url + '/api/decision/submit', { data: { revision: 1, draft_version: 0, submission_id: 'stale' } });
  expect(stale.status()).toBe(409);
  await page.locator('#history > summary').click();
  await expect(page.locator('#history-content')).toContainText('SQLite');
  await page.getByRole('button', { name: '提交裁定', exact: true }).click();
  expect((await next.result).items.map(item => item.status)).toEqual(['decided', 'pending', 'pending']);
});

test('a dirty browser refuses stale updates until the user loads the latest version', async ({ page, request, decisions }) => {
  await page.goto(decisions.initial.url);
  await expect(page.locator('.decision-card')).toHaveCount(3);
  await page.clock.install();
  await page.clock.pauseAt(new Date());
  await page.getByRole('radio', { name: 'SQLite', exact: true }).check();
  const changed = checklist(); changed.items[0].options[0].description = 'Changed option';
  const response = await request.put(decisions.initial.url + '/api/decision', { data: { checklist: changed, base_revision: 1 } });
  expect(response.ok()).toBeTruthy();
  await expect(page.locator('#error')).toContainText('当前修改仍保留');
  await expect(page.getByRole('button', { name: '提交裁定', exact: true })).toBeDisabled();
  await expect(page.getByRole('radio', { name: 'SQLite', exact: true })).toBeChecked();
  await page.getByRole('button', { name: '加载最新版本' }).click();
  await expect(page.locator('#revision')).toHaveText('第 2 轮');
  await expect(page.getByRole('radio', { name: 'SQLite', exact: true })).not.toBeChecked();
});

test('responsive themes, keyboard controls and sanitized Markdown', async ({ page, decisions }, testInfo) => {
  const input = checklist(); input.items[0].context = '<img src=x onerror="window.pwned=1"> [unsafe](javascript:alert(1)) **Safe Markdown** ' + 'longword'.repeat(100);
  const launched = await decisions.start(input);
  await page.goto(launched.url);
  await expect(page.locator('#item-0 strong').filter({ hasText: 'Safe Markdown' })).toBeVisible();
  await expect(page.locator('#items img, #items script, #items a[href^="javascript:"]')).toHaveCount(0);
  for (const theme of ['light', 'dark', 'system']) {
    await page.locator('#theme').selectOption(theme);
    for (const width of [375, 1440, 2560, 3440]) {
      await page.setViewportSize({ width, height: 1000 });
      await expect(async () => {
        const size = await page.locator('.workspace').evaluate(el => ({ width: el.getBoundingClientRect().width, overflow: document.documentElement.scrollWidth - innerWidth }));
        expect(size.overflow).toBeLessThanOrEqual(1);
        if (width >= 2560) expect(size.width).toBe(1920);
      }).toPass();
    }
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
  }
  await page.locator('#theme').selectOption('light');
  await page.screenshot({ path: testInfo.outputPath('decide-wide.png'), fullPage: true });
  await page.setViewportSize({ width: 375, height: 900 });
  const radio = page.getByRole('radio', { name: 'SQLite', exact: true });
  await radio.focus(); await page.keyboard.press('Space'); await expect(radio).toBeChecked();
  await page.locator('#item-0').getByRole('button', { name: '清除选择' }).click();
  await expect(radio).not.toBeChecked();
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: testInfo.outputPath('decide-mobile.png'), fullPage: true });
});
