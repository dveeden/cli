_empty :=
_space := $(_empty) $(empty)

# Gets added after the version
VERSION_POST ?=

# Auto bump by default
BUMP ?= auto
# If on master branch bump the minor by default
ifeq ($(RELEASE_BRANCH),$(MAIN_BRANCH))
DEFAULT_BUMP ?= minor
# Else bump the patch by default
else
DEFAULT_BUMP ?= patch
endif

VERSION ?= $(shell git rev-parse --is-inside-work-tree > /dev/null && git describe --tags --always --dirty)
ifneq (,$(findstring dirty,$(VERSION)))
VERSION := $(VERSION)-$(USER)
endif
CLEAN_VERSION ?= $(shell echo $(VERSION) | grep -Eo '([0-9]+\.){2}[0-9]+')
VERSION_NO_V := $(shell echo $(VERSION) | sed 's,^v,,' )

CI_SKIP ?= [ci skip]

ifeq ($(CLEAN_VERSION),$(_empty))
CLEAN_VERSION := 0.0.0
else
GIT_MESSAGES := $(shell git rev-parse --is-inside-work-tree > /dev/null && git log --pretty='%s' v$(CLEAN_VERSION)...HEAD 2>/dev/null | tr '\n' ' ')
endif

# If auto bump enabled, search git messages for bump hash
ifeq ($(BUMP),auto)
_auto_bump_msg := \(auto\)
ifneq (,$(findstring \#major,$(GIT_MESSAGES)))
BUMP := major
else ifneq (,$(findstring \#minor,$(GIT_MESSAGES)))
BUMP := minor
else ifneq (,$(findstring \#patch,$(GIT_MESSAGES)))
BUMP := patch
else
BUMP := $(DEFAULT_BUMP)
endif
endif

# Figure out what the next version should be
split_version := $(subst .,$(_space),$(CLEAN_VERSION))
ifeq ($(BUMP),major)
bump := $(shell expr $(word 1,$(split_version)) + 1)
BUMPED_CLEAN_VERSION := $(bump).0.0
else ifeq ($(BUMP),minor)
bump := $(shell expr $(word 2,$(split_version)) + 1)
BUMPED_CLEAN_VERSION := $(word 1,$(split_version)).$(bump).0
else ifeq ($(BUMP),patch)
bump := $(shell expr $(word 3,$(split_version)) + 1)
BUMPED_CLEAN_VERSION := $(word 1,$(split_version)).$(word 2,$(split_version)).$(bump)
endif

BUMPED_CLEAN_VERSION := $(BUMPED_CLEAN_VERSION)$(VERSION_POST)
BUMPED_VERSION := v$(BUMPED_CLEAN_VERSION)

RELEASE_SVG := <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="94" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="\#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="94" height="20" rx="3" fill="\#fff"/></clipPath><g clip-path="url(\#a)"><path fill="\#555" d="M0 0h49v20H0z"/><path fill="\#007ec6" d="M49 0h45v20H49z"/><path fill="url(\#b)" d="M0 0h94v20H0z"/></g><g fill="\#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="255" y="150" fill="\#010101" fill-opacity=".3" transform="scale(.1)" textLength="390">release</text><text x="255" y="140" transform="scale(.1)" textLength="390">release</text><text x="705" y="150" fill="\#010101" fill-opacity=".3" transform="scale(.1)" textLength="350">$(BUMPED_VERSION)</text><text x="705" y="140" transform="scale(.1)" textLength="350">$(BUMPED_VERSION)</text></g> </svg>

.PHONY: show-version
## Show version variables
show-version:
	@echo version: $(VERSION)
	@echo version no v: $(VERSION_NO_V)
	@echo clean version: $(CLEAN_VERSION)
	@echo version bump: $(BUMP) $(_auto_bump_msg)
	@echo bumped version: $(BUMPED_VERSION)
	@echo bumped clean version: $(BUMPED_CLEAN_VERSION)
	@echo version post append: $(VERSION_POST)
	@echo 'release svg: $(RELEASE_SVG)'

.PHONY: commit-release
commit-release:
	echo '$(RELEASE_SVG)' > release.svg
	git add release.svg

	git diff --exit-code --cached --name-status || \
	git commit -m "chore: $(BUMP) version bump $(BUMPED_VERSION) $(CI_SKIP)"
	git push

.PHONY: tag-release
tag-release:
	# Delete tag from the remote in case it already exists
	git tag -d $(BUMPED_VERSION) || true
	git push -d $(GIT_REMOTE_NAME) $(BUMPED_VERSION) || true

	# Add tag to the remote
	git tag $(BUMPED_VERSION)
	git push $(GIT_REMOTE_NAME) $(BUMPED_VERSION)
