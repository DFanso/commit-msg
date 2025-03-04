
# commit-msg

## Screenshot

Below is a sample execution of `commit-msg`:

![Commit-msg Screenshot](image.png)

Before running the application, ensure you have set the system environment variables.

## You can Use Gemini or Grok as the LLM to Generate Commit Messages

### Add `COMMIT_LLM` value are `gemini` or `grok`

### Add `GROK_API_KEY` to System Variables (if use grok)

### Add `GEMINI_API_KEY` to System Variables (if use gemini)



---

## Setup

To set up `commit-msg`, run the following command:

```bash
go run src/main.go --setup --path F:/Git/commit-msg --name commit-msg
```

---

## Usage

To run `commit-msg`, use:

```bash
go run src/main.go --path F:/Git/commit-msg
```

This will execute `commit-msg` in the current directory:

```bash
go run src/main.go .
```




