import { expect, test } from "@playwright/test";
import { startTestServer } from "./helpers/test-server";

test.describe("About & Licenses Modal E2E", () => {
	let server;

	test.beforeAll(async () => {
		server = await startTestServer();
	});

	test.afterAll(async () => {
		if (server) {
			await server.stop();
		}
	});

	test("opens About modal via header trigger, switches tabs, searches licenses, and closes", async ({
		page,
	}) => {
		await page.goto(server.url);

		// 1. Locate and click About & Licenses trigger in top navbar
		const trigger = page.locator('[data-testid="about-modal-trigger"]');
		await expect(trigger).toBeVisible();
		await trigger.click();

		// 2. Verify modal opens
		const modal = page.locator('[data-testid="about-modal"]');
		await expect(modal).toBeVisible();

		// 3. Verify About tab content
		await expect(page.getByRole("heading", { name: "Jokateko", exact: true })).toBeVisible();
		const modalLogo = modal.locator('[data-testid="about-modal-logo"]');
		await expect(modalLogo).toBeVisible();
		await expect.poll(() => modalLogo.evaluate((img) => img.naturalWidth)).toBeGreaterThan(0);
		await expect(page.getByText("Zero-Dependency, Tasks-as-Code Architecture")).toBeVisible();
		await expect(page.getByText("Project License (MIT)")).toBeVisible();
		await expect(modal.locator("pre").first()).toContainText("Permission is hereby granted, free of charge");

		// 4. Switch to Open Source Licenses tab
		const licensesTab = page.locator('[data-testid="licenses-tab-btn"]');
		await expect(licensesTab).toBeVisible();
		await licensesTab.click();

		// 5. Verify dependency list rendered: only main (direct) packages, not their transitive deps
		const items = modal.locator('[data-testid="license-package-item"]');
		await expect(items.first()).toBeVisible();
		for (const name of ["mermaid", "preact", "lucide-preact", "github.com/yuin/goldmark"]) {
			await expect(items.filter({ hasText: name }).first()).toBeVisible();
		}
		await expect(items.filter({ hasText: "dompurify" })).toHaveCount(0);
		await expect(items.filter({ hasText: "golang.org/x/sys" })).toHaveCount(0);

		// 6. Test search filter
		const searchInput = modal.locator('[data-testid="license-search-input"]');
		await expect(searchInput).toBeVisible();
		await searchInput.fill("mermaid");

		// Filtered list should now include mermaid
		await expect(modal.locator('[data-testid="license-package-item"]', { hasText: "mermaid" }).first()).toBeVisible();

		// Verify external repository link is normalized to https:// and opens in new tab
		const repoLink = modal.locator('[data-testid="license-package-item"] a[title="Open repository"]').first();
		if (await repoLink.count() > 0) {
			const href = await repoLink.getAttribute("href");
			expect(href).toMatch(/^https:\/\//);
			expect(await repoLink.getAttribute("target")).toBe("_blank");
			expect(await repoLink.getAttribute("rel")).toContain("noopener");
		}

		// 7. Test ecosystem filter buttons and verify type declarations and the Bun build plugin are excluded
		await searchInput.clear();
		await searchInput.fill("@types/");
		await expect(modal.locator('[data-testid="license-package-item"]')).toHaveCount(0);
		await searchInput.clear();
		await searchInput.fill("bun-plugin-tailwind");
		await expect(modal.locator('[data-testid="license-package-item"]')).toHaveCount(0);
		await searchInput.clear();

		const goFilterBtn = modal.locator('[data-testid="filter-go-btn"]');
		await goFilterBtn.click();
		await expect(modal.locator('[data-testid="license-package-item"]', { hasText: "go-toml" }).first()).toBeVisible();

		// 8. Test viewing license text
		const viewTextBtn = modal.locator('[data-testid="license-view-text-btn"]').first();
		await viewTextBtn.click();
		await expect(modal.locator("pre").first()).toBeVisible();

		// 9. Close modal via close button
		const closeBtn = modal.locator('[data-testid="about-modal-close"]');
		await closeBtn.click();
		await expect(modal).not.toBeVisible();

		// 10. Re-open via Footer Licenses trigger (should open directly to Open Source Licenses tab)
		const footerTrigger = page.locator('[data-testid="footer-licenses-trigger"]');
		await expect(footerTrigger).toBeVisible();
		await footerTrigger.click();
		await expect(modal).toBeVisible();

		// Verify licenses search input is immediately visible (Licenses tab active)
		await expect(modal.locator('[data-testid="license-search-input"]')).toBeVisible();

		await page.keyboard.press("Escape");
		await expect(modal).not.toBeVisible();
	});

	test("footer shows the logo next to the project link", async ({ page }) => {
		await page.goto(server.url);

		const logo = page.locator('[data-testid="app-footer"] [data-testid="footer-logo"]');
		await expect(logo).toBeVisible();
		// The SVG data URI decoded (a CSP block or bad markup leaves naturalWidth at 0)
		await expect.poll(() => logo.evaluate((img) => img.naturalWidth)).toBeGreaterThan(0);
		await expect(logo).toHaveAttribute("alt", "");
	});
});
