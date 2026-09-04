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
		await expect(lightbox.locator("svg")).toBeVisible();

		const closeLightbox = page.locator('[data-testid="mermaid-lightbox-close"]');
		await closeLightbox.click();
		await expect(lightbox).not.toBeVisible();

		// 7. Verify Task Detail Modal Maximize / Restore button
		const maximizeBtn = modal.locator('[data-testid="task-modal-maximize-btn"]');
		await expect(maximizeBtn).toBeVisible();
		await maximizeBtn.click();

		const modalBox = modal.locator(".modal-box");
		await expect(modalBox).toHaveClass(/max-w-\[96vw\]/);

		await maximizeBtn.click();
		await expect(modalBox).toHaveClass(/max-w-2xl/);
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
