VENV := .venv
MKDOCS := $(VENV)/bin/mkdocs

.DEFAULT_GOAL := docs
.PHONY: venv docs docs-build clean

venv: $(MKDOCS)

$(MKDOCS): requirements.txt
	python3 -m venv $(VENV)
	$(VENV)/bin/pip install -q --upgrade pip
	$(VENV)/bin/pip install -q -r requirements.txt
	@touch $@

docs: $(MKDOCS)
	$(MKDOCS) serve

docs-build: $(MKDOCS)
	$(MKDOCS) build --strict

clean:
	rm -rf site
