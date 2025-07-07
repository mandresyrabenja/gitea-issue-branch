# Gitea Issue → Branch Action

**Create a Git branch automatically from a Gitea issue, then attach that branch back to the issue via the Gitea REST API.**  

---

## What it does

1. **Reads issue data** from the workflow event.  
2. **Maps issue labels → Git Flow prefix** (`feature`, `bugfix`, `hotfix`).  
3. **Creates a branch** named `<prefix>/us-<ISSUE_NUMBER>` from `origin/develop` (configurable).  
4. **Pushes the branch** to the remote repository.  
5. **Updates the issue** so its “linked branch” points to the one it just created.

---

## Environment variables

| Variable name | Required  | Example value                | Purpose                                        |
|---------------|-----------|------------------------------|------------------------------------------------|
| `GITEA_URL`   | *         | `https://git.example.com`    | Base URL of the Gitea instance                 |
| `GITEA_TOKEN` | *         | `ghp_xxx…`                   | Personal access token with **repo** scope      |
| `REPO_OWNER`  | *         | `acme`                       | Repository owner (user or org)                 |
| `REPO_NAME`   | *         | `widgets`                    | Repository name                                |
| `ISSUE_NUMBER`| *         | `42`                         | Issue number to work from                      |
| `ISSUE_TITLE` | *         | `Add Dark Mode`              | Issue title (used for logging only)            |
| `ISSUE_LABELS`| –         | `bug, ui`                    | Comma-separated list of labels                 |
| `BASE_REF`    | –         | `main`                       | Base branch/commit SHA (default **develop**)   |

> **Label → prefix mapping**

| Label          | Prefix   |
|----------------|----------|
| `enhancement`  | `feature`|
| `bug`          | `hotfix` |
| `invalid`      | `bugfix` |
| _anything else_| `feature`|

---

## Quick start (Gitea Actions)

Exactly the same YAML—`act_runner` builds the Dockerfile automatically:

```yaml
      - uses: you/gitea-create-branch-action@v1
        env:
          GITEA_URL:   https://git.example.com
          GITEA_TOKEN: ${{ secrets.GITEA_TOKEN }}
          REPO_OWNER:  acme
          REPO_NAME:   widgets
          ISSUE_NUMBER: ${{ gitea.event.issue.number }}
          ISSUE_TITLE:  ${{ gitea.event.issue.title }}
          ISSUE_LABELS: ${{ join(gitea.event.issue.labels.*.name, ',') }}
```

---

## Branch-naming rules

```
<prefix>/us-<ISSUE_NUMBER>
```

* `prefix` = **feature**, **bugfix**, or **hotfix** (based on label table above)
* Example: `feature/us-123`, `hotfix/us-7`

---

## Local development

```bash
# build and run the container locally
docker build -t gitea-issue-branch-action .
docker run --rm \
  -e GITEA_URL=https://git.example.com \
  -e GITEA_TOKEN=xxx \
  -e REPO_OWNER=acme \
  -e REPO_NAME=widgets \
  -e ISSUE_NUMBER=42 \
  -e ISSUE_TITLE="Example" \
  -e ISSUE_LABELS="enhancement" \
  gitea-issue-branch-action
```

Or use [`act`](https://github.com/nektos/act) to execute the full workflow.

---

## 📄 License

Distributed under the MIT License—see [`LICENSE`](./LICENSE) for details.

Feel free to open issues or pull requests to enhance the action!
