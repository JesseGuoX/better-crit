import { test, expect } from '@playwright/test';
import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { addComment, clearAllComments, getMdPath, goSection, loadPage, mdSection } from './helpers';

test.describe('Markdown reading layout', () => {
  test.beforeEach(async ({ page, request }) => {
    await clearAllComments(request);
    await page.context().addCookies([{
      name: 'crit-settings',
      value: encodeURIComponent(JSON.stringify({ width: 'wide' })),
      domain: 'localhost', path: '/',
    }]);
    await loadPage(page);
  });

  test('prose follows the available canvas width across screen sizes and themes', async ({ page }) => {
    const doc = mdSection(page).locator('.markdown-document');
    await expect(doc).toBeVisible();
    for (const colorScheme of ['light', 'dark'] as const) {
      await page.emulateMedia({ colorScheme });
      for (const width of [375, 1440, 1920, 2560, 3440]) {
        await page.setViewportSize({ width, height: 1000 });
        await expect(async () => {
          const sizes = await doc.evaluate(el => {
            const paragraph = el.querySelector('.line-content > p')!;
            const line = paragraph.parentElement!;
            const style = getComputedStyle(line);
            return {
              canvas: el.getBoundingClientRect().width,
              prose: paragraph.getBoundingClientRect().width,
              available: line.getBoundingClientRect().width - parseFloat(style.paddingLeft) - parseFloat(style.paddingRight),
              overflow: document.documentElement.scrollWidth - window.innerWidth,
            };
          });
          expect(sizes.canvas).toBeLessThanOrEqual(1920);
          expect(Math.abs(sizes.prose - sizes.available)).toBeLessThanOrEqual(1);
          expect(sizes.overflow).toBeLessThanOrEqual(1);
          if (width >= 2560) {
            expect(sizes.canvas).toBeGreaterThan(1280);
            expect(sizes.prose).toBeGreaterThan(1280);
          }
        }).toPass();
      }
    }
    // The Markdown canvas must not widen full-file code documents.
    await expect(goSection(page).locator('.code-document')).toHaveCSS('max-width', '1280px');
  });

  test('width selection persists and headings remain aligned with their source gutters', async ({ page }) => {
    await page.setViewportSize({ width: 2560, height: 1200 });
    const doc = mdSection(page).locator('.markdown-document');
    await page.locator('#settingsToggle').click();
    for (const [choice, width] of [['compact', '840px'], ['default', '1040px'], ['wide', '1920px']]) {
      await page.locator(`[data-settings-width="${choice}"]`).click();
      await expect(doc).toHaveCSS('max-width', width);
    }
    await page.reload();
    await expect(doc).toHaveCSS('max-width', '1920px');
    const offsets = await doc.locator('.line-block:has(> .line-content > h2)').evaluateAll(rows =>
      rows.map(row => Math.abs(row.querySelector('.line-gutter')!.getBoundingClientRect().top
        - row.querySelector('h2')!.getBoundingClientRect().top)),
    );
    expect(offsets.length).toBeGreaterThan(0);
    for (const offset of offsets) expect(offset).toBeLessThanOrEqual(1);
  });

  test('open sidebars and resized panels leave comments aligned with the document', async ({ page, request }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await addComment(request, await getMdPath(request), 1, 'Review the reading layout');
    await loadPage(page);
    await page.locator('.comment-count-icon').first().click();
    await expect(page.locator('#commentsPanel')).toBeVisible();
    const handle = page.locator('#fileTreeResizer');
    await handle.scrollIntoViewIfNeeded();
    const box = (await handle.boundingBox())!;
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();
    await page.mouse.move(box.x + 100, box.y + box.height / 2, { steps: 10 });
    await page.mouse.up();
    const doc = mdSection(page).locator('.markdown-document');
    await expect(doc.locator('.comment-card')).toBeVisible();
    await expect(async () => {
      const layout = await doc.evaluate(el => {
        const heading = el.querySelector('h1')!.getBoundingClientRect();
        const comment = el.querySelector('.comment-card')!.getBoundingClientRect();
        return { offset: Math.abs(heading.left - comment.left),
          overflow: document.documentElement.scrollWidth - window.innerWidth };
      });
      expect(layout.offset).toBeLessThanOrEqual(1);
      expect(layout.overflow).toBeLessThanOrEqual(1);
    }).toPass();
  });

  test('long Markdown and round controls stay within a narrow viewport after refresh', async ({ page, request }) => {
    const session = await (await request.get('/api/session')).json();
    const file = join(session.cwd, await getMdPath(request));
    const original = readFileSync(file, 'utf8');
    try {
      writeFileSync(file, original + '\n## 移动宽屏检查\n\n' + '中文 English 长段落应当随窗口宽度自然换行。'.repeat(20)
        + '\n\n| Long identifier | 中文说明 |\n| --- | --- |\n| ' + 'identifier_'.repeat(50)
        + ' | 很宽的表格应在自身容器内滚动。 |\n\n```js\nconst value = "' + 'long-value-'.repeat(60) + '";\n```\n');
      await expect(async () => {
        await loadPage(page);
        await expect(mdSection(page).locator('h2').last()).toHaveText('移动宽屏检查');
      }).toPass();
      await request.post('/api/round-complete');
      await loadPage(page);
      await page.setViewportSize({ width: 375, height: 812 });
      await expect(mdSection(page).locator('.markdown-document')).toBeVisible();
      await expect(page.locator('#diffToggle')).toBeVisible();
      await expect(page.locator('#finishBtn')).toBeInViewport();
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(1);
      await page.reload();
      await expect(mdSection(page).locator('.markdown-document')).toBeVisible();
      await expect.poll(() => page.evaluate(() => Math.abs(
        parseFloat(document.documentElement.style.getPropertyValue('--header-height'))
        - document.querySelector('.header')!.getBoundingClientRect().height,
      ))).toBeLessThanOrEqual(1);
    } finally {
      writeFileSync(file, original);
      await request.post('/api/round-complete');
    }
  });
});
