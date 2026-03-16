SHELL := /bin/sh

.PHONY: publish

publish:
	@tag="$${TAG}"; \
	module_path="$$(go list -m -f '{{.Path}}')"; \
	if ! git diff-index --quiet HEAD --; then \
		echo "Working tree has uncommitted changes. Commit them before publishing."; \
		exit 1; \
	fi; \
	if [ -z "$$tag" ]; then \
		printf "Enter new tag: "; \
		read tag; \
	fi; \
	if [ -z "$$tag" ]; then \
		echo "Tag is required."; \
		exit 1; \
	fi; \
	if [ -z "$$module_path" ]; then \
		echo "Could not determine Go module path."; \
		exit 1; \
	fi; \
	git tag "$$tag" && git push origin "$$tag"; \
	if command -v curl >/dev/null 2>&1; then \
		curl --fail --silent --show-error -X POST "https://pkg.go.dev/fetch/$$module_path@$$tag"; \
		echo; \
	else \
		echo "Tag pushed, but pkg.go.dev fetch was skipped because curl is not installed."; \
	fi; \
	if command -v gh >/dev/null 2>&1; then \
		gh release create "$$tag" --generate-notes; \
	else \
		echo "Tag pushed, but no GitHub release was created because gh is not installed."; \
	fi
