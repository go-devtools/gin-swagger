const { test, expect } = require('./fixtures.cjs');

// Render actual offline documentation without enabling request execution by default.
test('safe defaults, display names, enum labels, and definition selection', async ({ page, request }, testInfo) => {
 await page.goto('/safe/docs/?url=https://outside.invalid/spec.json');
 await expect(page).toHaveTitle('Browser contract');
 await expect(page.locator('.info .title')).toContainText('Browser API');
 const operation = page.locator('.opblock').filter({ hasText: '/submit' });
 await expect(operation).toBeVisible();
 await expect(operation.getByRole('button', { name: 'Try it out', exact: true })).toHaveCount(0);
 await expect(operation.getByRole('button', { name: 'Execute', exact: true })).toHaveCount(0);
 await operation.getByRole('tab', { name: 'Schema', exact: true }).first().click();
 await expect(operation.getByLabel('Allowed values').first()).toContainText('"admin" - Administrator');
 await expect(operation.getByLabel('Allowed values').first()).toContainText('"editor" - Editor');
 await expect(page.locator('.models')).not.toContainText(/_[a-f0-9]{12}|_opaque/);
 await expect(page.getByPlaceholder('Filter by tag')).toHaveCount(0);
 await page.screenshot({ path: testInfo.outputPath('safe-desktop.png'), fullPage: true });
 await page.locator('.topbar select').selectOption({ label: 'Reference' });
 await expect(page.locator('.info .title')).toContainText('Reference');
 await page.locator('.topbar select').selectOption({ label: 'All endpoints' });
 await expect(page.locator('.info .title')).toContainText('Browser API');
 await page.setViewportSize({ width: 390, height: 844 });
 await expect(page.locator('.info .title')).toBeVisible();
 await page.screenshot({ path: testInfo.outputPath('safe-mobile.png'), fullPage: true });
 const state = await (await request.get('/state')).json();
 expect(state.requests).toBe(0);
});

// Send a harmless local request only after explicit configuration and Bearer authorization.
test('explicit submission sends the Bearer token and typed JSON', async ({ page, request }, testInfo) => {
 await page.goto('/enabled/docs/');
 await expect(page.locator('.info .title')).toContainText('Browser API');
 await page.locator('.auth-wrapper').getByRole('button', { name: 'Authorize' }).click();
 const dialog = page.locator('.dialog-ux');
 await expect(dialog).toContainText('BearerAuth');
 await expect(dialog.locator('input')).toHaveCount(1);
 await dialog.locator('input').fill('ci-demo-token');
 await dialog.getByRole('button', { name: 'Apply credentials', exact: true }).click();
 await dialog.getByRole('button', { name: 'Close', exact: true }).click();
 const operation = page.locator('.opblock').filter({ hasText: '/submit' });
 await operation.locator('textarea').fill('{"Role":"admin"}');
 const responsePromise = page.waitForResponse(response => response.url().endsWith('/submit') && response.request().method() === 'POST');
 await operation.getByRole('button', { name: 'Execute', exact: true }).click();
 const response = await responsePromise;
 expect(response.status()).toBe(200);
 expect(await response.json()).toEqual({ Role: 'admin' });
 const state = await (await request.get('/state')).json();
 expect(state.requests).toBe(1);
 expect(state.authorization).toBe('Bearer ci-demo-token');
 await page.screenshot({ path: testInfo.outputPath('explicit-submit.png'), fullPage: true });
});

// Consume the shared renderer from the pinned core, preserving native metadata and default-disabled submission.
test('Gin mounting serves shared native compatibility diagnostics', async ({ page, request }) => {
 await page.goto('/metadata/docs/');
 await expect(page.locator('.info .title')).toContainText('Browser API');
 const notes = page.getByRole('region', { name: 'Swagger UI compatibility' });
 await expect(notes).toBeVisible();
 await notes.locator('summary').click();
 await expect(notes).toContainText('summary: "Browser examples", parent: "Platform", kind: "nav"');
 const original = await (await request.get('/metadata/docs/groups/all.json')).json();
 expect(original.openapi).toBe('3.2.0');
 expect(original.tags.find(tag => tag.name === 'Browser')).toMatchObject({ summary: 'Browser examples', parent: 'Platform', kind: 'nav' });
 expect(await page.evaluate(() => window.ui.specSelectors.specJson().toJS())).toEqual(original);
 const diagnostics = await page.evaluate(() => window.ui.fn.openapiUICompatibility(window.ui.specSelectors.specJson().toJS()).diagnostics);
 expect(diagnostics.map(d => d.code)).toEqual(['openapi.ui.tagMetadata', 'openapi.ui.tagMetadata']);
 const resource = await request.get('/metadata/docs/native-compatibility.js');
 expect(resource.status()).toBe(200);
 expect(resource.headers()['content-type']).toContain('javascript');
 expect(resource.headers()['x-content-type-options']).toBe('nosniff');
 await expect(page.getByRole('button', { name: 'Execute', exact: true })).toHaveCount(0);
 await expect(page.getByRole('button', { name: 'Try it out', exact: true })).toHaveCount(0);
});
