SHELL := /bin/sh

.PHONY: publish

publish:
	@tag="$${TAG}"; \
	if [ -z "$$tag" ]; then \
		printf "Enter new tag: "; \
		read tag; \
	fi; \
	if [ -z "$$tag" ]; then \
		echo "Tag is required."; \
		exit 1; \
	fi; \
	git tag "$$tag" && git push origin "$$tag"
