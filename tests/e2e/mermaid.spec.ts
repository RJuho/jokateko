import { expect, test } from "@playwright/test";
import { startTestServer } from "./helpers/test-server";

test.describe("Inlined Mermaid Diagram Rendering E2E", () => {
	let server;

	test.beforeAll(async () => {
		server = await startTestServer({
			customTasks: [
				{
					id: "260904-mermaid-valid-diagram",
					title: "Task with Valid Mermaid Diagram",
					status: "in_progress",
					priority: "high",
					tags: ["feature", "frontend"],
					summary: "Valid mermaid diagram flowchart test",
					body: [
						"# Valid Diagram Spec",
						"",
						"Here is an architectural diagram:",
						"",
						"```mermaid",
						"graph TD",
						"    Client[Web Client] --> Gateway[API Gateway]",
						"    Gateway --> DB[(SQLite DB)]",
						"```",
						"",
						"## Acceptance Criteria",
						"- [ ] Render diagram correctly",
					].join("\n"),
				},
				{
					id: "260904-mermaid-invalid-diagram",
					title: "Task with Invalid Mermaid Diagram",
					status: "in_progress",
					priority: "low",
					tags: ["testing"],
					summary: "Invalid mermaid diagram fallback test",
					body: [
						"# Invalid Diagram Spec",
						"",
						"```mermaid",
						"this is not valid mermaid syntax %%% $$$ !!!",
						"```",
						"",
						"## Acceptance Criteria",
						"- [ ] Fallback test",
					].join("\n"),
				},
			],
			customStrategies: [
				{
					id: "system-architecture",
					title: "System Architecture Flow",
					tier: 1,
					summary: "Architecture diagram in strategy",
					tags: ["architecture"],
					body: [
						"# System Architecture Flow",
						"",
						"```mermaid",
						"graph LR",
						"    App[Application] --> Broker[Message Broker]",
						"    Broker --> Worker[Background Worker]",
						"```",
					].join("\n"),
				},
			],
			customGlossary: [
				{
					id: "event-stream",
					title: "Event Stream",
					summary: "Glossary term with mermaid diagram",
					tags: ["events"],
					body: [
						"# Event Stream Flow",
						"",
						"```mermaid",
						"sequenceDiagram",
						"    Alice->>Bob: Hello Bob",
						"    Bob-->>Alice: Hi Alice",
						"```",
					].join("\n"),
				},
			],
		});
	});

	test.afterAll(async () => {
		if (server) {
			await server.stop();
		}
	});

	test("renders a valid mermaid diagram with zoom/fullscreen toolbar inside task modal", async ({
		page,
	}) => {
		await page.goto(server.url);

		// 1. Locate and open the task modal
		const card = page.locator(
			'[data-testid="task-card-260904-mermaid-valid-diagram"]',
		);
		await expect(card).toBeVisible();
		await card.click();

		const modal = page.locator('[data-testid="task-detail-modal"]');
		await expect(modal).toBeVisible();

		// 2. Locate the rendered mermaid diagram container
		const diagram = modal.locator('[data-testid="mermaid-diagram"]');
		await expect(diagram).toBeVisible();

		// 3. Verify that an SVG element is rendered inside .mermaid-inner
		const svg = diagram.locator(".mermaid-inner svg");
		await expect(svg).toBeVisible();

		// 4. Verify the diagram contains text from the mermaid nodes
		await expect(svg).toContainText("Web Client");
		await expect(svg).toContainText("API Gateway");
		await expect(svg).toContainText("SQLite DB");

		// 5. Verify toolbar controls: Zoom In, Zoom Out, Zoom Reset
		const zoomIn = diagram.locator('[data-testid="mermaid-zoom-in"]');
		const zoomReset = diagram.locator('[data-testid="mermaid-zoom-reset"]');
		await expect(zoomIn).toBeVisible();
		await expect(zoomReset).toHaveText("100%");

		await zoomIn.click();
		await expect(zoomReset).toHaveText("125%");

		const zoomOut = diagram.locator('[data-testid="mermaid-zoom-out"]');
		await zoomOut.click();
		await expect(zoomReset).toHaveText("100%");

		// 6. Verify Fullscreen / Lightbox expand
		const expandBtn = diagram.locator('[data-testid="mermaid-expand-btn"]');
		await expect(expandBtn).toBeVisible();
		await expandBtn.click();

		const lightbox = page.locator('[data-testid="mermaid-lightbox-dialog"]');
		await expect(lightbox).toBeVisible();
		await expect(lightbox.locator(".mermaid-inner svg")).toBeVisible();

		const closeLightbox = page.locator('[data-testid="mermaid-lightbox-close"]');
		await closeLightbox.click();
		await expect(lightbox).not.toBeVisible();

		// 6b. Escape closes only the lightbox, task modal stays open underneath
		await expandBtn.click();
		await expect(lightbox).toBeVisible();
		await page.keyboard.press("Escape");
		await expect(lightbox).not.toBeVisible();
		await expect(modal).toBeVisible();

		// 7. Verify Task Detail Modal Maximize / Restore button
		const maximizeBtn = modal.locator('[data-testid="task-modal-maximize-btn"]');
		await expect(maximizeBtn).toBeVisible();
		await maximizeBtn.click();

		const modalBox = modal.locator(".modal-box");
		await expect(modalBox).toHaveClass(/max-w-\[96vw\]/);

		await maximizeBtn.click();
		await expect(modalBox).toHaveClass(/max-w-2xl/);
	});

	test("pans by mouse drag and zooms with Ctrl+wheel, keyboard and buttons", async ({
		page,
	}) => {
		await page.goto(`${server.url}/#task/260904-mermaid-valid-diagram`);
		const diagram = page.locator('[data-testid="task-detail-modal"] [data-testid="mermaid-diagram"]');
		const viewport = diagram.locator(".mermaid-viewport");
		const svg = diagram.locator(".mermaid-inner svg");
		const readout = diagram.locator('[data-testid="mermaid-zoom-reset"]');
		await expect(svg).toBeVisible();

		const metrics = () =>
			viewport.evaluate((vp) => {
				const svgRect = vp.querySelector(".mermaid-inner svg").getBoundingClientRect();
				const vpRect = vp.getBoundingClientRect();
				return {
					scrollLeft: vp.scrollLeft,
					scrollTop: vp.scrollTop,
					overflowX: vp.scrollWidth - vp.clientWidth,
					overflowY: vp.scrollHeight - vp.clientHeight,
					svgWidth: svgRect.width,
					// with scroll at 0, the diagram's left/top edge must be visible (not cut off)
					leftGap: svgRect.left - vpRect.left + vp.scrollLeft,
					topGap: svgRect.top - vpRect.top + vp.scrollTop,
				};
			});

		// 1. Zooming resizes the diagram so the scroll area grows (not a CSS transform)
		const fitWidth = (await metrics()).svgWidth;
		for (let i = 0; i < 5; i++) await diagram.locator('[data-testid="mermaid-zoom-in"]').click();
		await expect(readout).toHaveText("305%");
		const zoomed = await metrics();
		expect(zoomed.svgWidth).toBeGreaterThan(fitWidth * 2.9);
		expect(zoomed.overflowX + zoomed.overflowY).toBeGreaterThan(0);

		// 2. Every edge stays reachable: scrolled to the origin, nothing is cut off on the left/top
		await viewport.evaluate((vp) => vp.scrollTo(0, 0));
		const origin = await metrics();
		expect(origin.leftGap).toBeGreaterThanOrEqual(0);
		expect(origin.topGap).toBeGreaterThanOrEqual(0);

		// 3. Mouse drag pans the diagram
		const box = await viewport.boundingBox();
		await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
		await page.mouse.down();
		await page.mouse.move(box.x + box.width / 2 - 80, box.y + box.height / 2 - 60, { steps: 5 });
		await page.mouse.up();
		const dragged = await metrics();
		expect(dragged.scrollLeft + dragged.scrollTop).toBeGreaterThan(0);

		// 4. Keyboard: 0 resets, + zooms in, - zooms out (viewport is focusable)
		await viewport.focus();
		await page.keyboard.press("0");
		await expect(readout).toHaveText("100%");
		await page.keyboard.press("+");
		await expect(readout).toHaveText("125%");
		await page.keyboard.press("-");
		await expect(readout).toHaveText("100%");

		// 5. Ctrl + wheel zooms around the cursor; plain wheel does not zoom
		await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
		await page.mouse.wheel(0, 100);
		await expect(readout).toHaveText("100%");
		// the plain wheel may have scrolled the modal (native scroll chaining): re-locate the viewport
		await viewport.scrollIntoViewIfNeeded();
		const box2 = await viewport.boundingBox();
		await page.mouse.move(box2.x + box2.width / 2, box2.y + box2.height / 2);
		await page.keyboard.down("Control");
		await page.mouse.wheel(0, -300);
		await page.keyboard.up("Control");
		await expect(readout).not.toHaveText("100%");
		expect((await metrics()).svgWidth).toBeGreaterThan(fitWidth * 1.1);

		// 6. Fullscreen lightbox fits the whole diagram and has its own zoom controls
		await diagram.locator('[data-testid="mermaid-expand-btn"]').click();
		const lightbox = page.locator('[data-testid="mermaid-lightbox-dialog"]');
		const lbViewport = lightbox.locator(".mermaid-viewport");
		await expect(lightbox.locator(".mermaid-inner svg")).toBeVisible();
		await expect(lightbox.locator('[data-testid="mermaid-zoom-reset"]')).toHaveText("100%");
		const fits = await lbViewport.evaluate(
			(vp) => vp.scrollWidth <= vp.clientWidth + 1 && vp.scrollHeight <= vp.clientHeight + 1,
		);
		expect(fits).toBe(true);
		await lightbox.locator('[data-testid="mermaid-zoom-in"]').click();
		await expect(lightbox.locator('[data-testid="mermaid-zoom-reset"]')).toHaveText("125%");
		await page.keyboard.press("Escape");
		await expect(lightbox).not.toBeVisible();
	});

	test("re-renders diagram seamlessly when theme is toggled", async ({
		page,
	}) => {
		await page.goto(server.url);

		const card = page.locator(
			'[data-testid="task-card-260904-mermaid-valid-diagram"]',
		);
		await card.click();

		const modal = page.locator('[data-testid="task-detail-modal"]');
		await expect(modal).toBeVisible();

		const diagram = modal.locator('[data-testid="mermaid-diagram"]');
		await expect(diagram).toBeVisible();
		const svg = diagram.locator(".mermaid-inner svg");
		await expect(svg).toBeVisible();

		// Toggle theme attribute
		await page.evaluate(() => {
			const current = document.documentElement.getAttribute("data-theme") || "winter";
			const next = current === "sunset" ? "winter" : "sunset";
			document.documentElement.setAttribute("data-theme", next);
		});

		// Ensure diagram is still visible and has not fallen back to raw code
		await expect(diagram).toBeVisible();
		await expect(svg).toBeVisible();
		await expect(modal.locator("pre code.language-mermaid")).toHaveCount(0);
	});

	test("gracefully retains raw pre/code block when Mermaid syntax fails to parse", async ({
		page,
	}) => {
		await page.goto(server.url);

		// 1. Locate and open the invalid task modal
		const card = page.locator(
			'[data-testid="task-card-260904-mermaid-invalid-diagram"]',
		);
		await expect(card).toBeVisible();
		await card.click();

		const modal = page.locator('[data-testid="task-detail-modal"]');
		await expect(modal).toBeVisible();

		// 2. Verify no diagram container exists
		const diagram = modal.locator('[data-testid="mermaid-diagram"]');
		await expect(diagram).toHaveCount(0);

		// 3. Verify original pre/code block is preserved
		const codeBlock = modal.locator("pre code.language-mermaid");
		await expect(codeBlock).toBeVisible();
		await expect(codeBlock).toContainText(
			"this is not valid mermaid syntax %%% $$$ !!!",
		);
	});

	test("renders mermaid diagram with toolbar in strategies view and responds to theme change", async ({
		page,
	}) => {
		await page.goto(`${server.url}/#strategies`);

		// 1. Locate strategy card and expand it
		const card = page.locator(
			'[data-testid="strategy-card-system-architecture"]',
		);
		await expect(card).toBeVisible();

		// Click card to open if not already open
		const expandButton = card.locator(
			'button[aria-label="Strategy: System Architecture Flow"]',
		);
		await expandButton.click();

		// 2. Verify Mermaid diagram is rendered inside the strategy body
		const diagram = card.locator('[data-testid="mermaid-diagram"]');
		await expect(diagram).toBeVisible();

		const svg = diagram.locator(".mermaid-inner svg");
		await expect(svg).toBeVisible();
		await expect(svg).toContainText("Application");
		await expect(svg).toContainText("Message Broker");
		await expect(svg).toContainText("Background Worker");

		// 3. Verify zoom toolbar controls
		const zoomIn = diagram.locator('[data-testid="mermaid-zoom-in"]');
		const zoomReset = diagram.locator('[data-testid="mermaid-zoom-reset"]');
		await expect(zoomIn).toBeVisible();
		await expect(zoomReset).toHaveText("100%");

		await zoomIn.click();
		await expect(zoomReset).toHaveText("125%");

		// 4. Toggle theme attribute and ensure diagram stays rendered
		await page.evaluate(() => {
			const current =
				document.documentElement.getAttribute("data-theme") || "winter";
			const next = current === "sunset" ? "winter" : "sunset";
			document.documentElement.setAttribute("data-theme", next);
		});

		await expect(diagram).toBeVisible();
		await expect(svg).toBeVisible();
		await expect(card.locator("pre code.language-mermaid")).toHaveCount(0);
	});

	test("renders mermaid diagram with toolbar in glossary view and responds to theme change", async ({
		page,
	}) => {
		await page.goto(`${server.url}/#glossary`);

		// 1. Locate glossary card and expand it
		const card = page.locator('[data-testid="glossary-card-event-stream"]');
		await expect(card).toBeVisible();

		// Click card summary or title button to open
		const expandButton = card.locator(
			'button[aria-label="Glossary term: Event Stream"]',
		);
		await expandButton.click();

		// 2. Verify Mermaid diagram is rendered inside the glossary body
		const diagram = card.locator('[data-testid="mermaid-diagram"]');
		await expect(diagram).toBeVisible();

		const svg = diagram.locator(".mermaid-inner svg");
		await expect(svg).toBeVisible();
		await expect(svg).toContainText("Alice");
		await expect(svg).toContainText("Bob");

		// 3. Verify zoom toolbar controls
		const zoomIn = diagram.locator('[data-testid="mermaid-zoom-in"]');
		const zoomReset = diagram.locator('[data-testid="mermaid-zoom-reset"]');
		await expect(zoomIn).toBeVisible();
		await expect(zoomReset).toHaveText("100%");

		await zoomIn.click();
		await expect(zoomReset).toHaveText("125%");

		// 4. Toggle theme attribute and ensure diagram stays rendered
		await page.evaluate(() => {
			const current =
				document.documentElement.getAttribute("data-theme") || "winter";
			const next = current === "sunset" ? "winter" : "sunset";
			document.documentElement.setAttribute("data-theme", next);
		});

		await expect(diagram).toBeVisible();
		await expect(svg).toBeVisible();
		await expect(card.locator("pre code.language-mermaid")).toHaveCount(0);
	});
});
