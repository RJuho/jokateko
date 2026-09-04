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
		});
	});

	test.afterAll(async () => {
		if (server) {
			await server.stop();
		}
	});

	test("renders a valid mermaid diagram as an interactive SVG element inside task modal", async ({
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

		// 3. Verify that an SVG element is rendered inside the container
		const svg = diagram.locator("svg");
		await expect(svg).toBeVisible();

		// 4. Verify the diagram contains text from the mermaid nodes
		await expect(svg).toContainText("Web Client");
		await expect(svg).toContainText("API Gateway");
		await expect(svg).toContainText("SQLite DB");
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
});
