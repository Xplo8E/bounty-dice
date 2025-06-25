# Bounty Dice

Bounty Dice is not just a tool; it's a training system. It's designed for bug bounty hunters who find themselves paralyzed by choice, endlessly scrolling through programs without ever committing to one. Its purpose is to enforce focus, cultivate discipline, and turn the overwhelming chaos of bug bounty hunting into a structured, gamified mission.

## The Philosophy: Focus is a Superpower

In a world of infinite targets, the most valuable asset is not a fancy tool or a secret methodology—it's **unwavering focus**. The modern bug bounty landscape encourages a "grass is always greener" mindset, leading hunters to jump from program to program, only scratching the surface. This rarely leads to significant findings.

Bounty Dice is built on a simple premise: **deep, prolonged engagement with a single target yields better results.** By committing to one program for a set period, you move beyond the low-hanging fruit that everyone finds in the first 48 hours. You start to understand the application's logic, its history, its architecture, and its blind spots. This is where the critical, high-impact vulnerabilities are found.

## The Psychology of the Dice

Every feature in Bounty Dice is intentionally designed to combat indecision and build the mental muscle of a dedicated hunter.

*   **The Roll:** The initial roll is a leap of faith. By accepting a randomly assigned target (within your chosen filters), you are outsourcing the single most paralyzing decision. Your job is no longer to *choose* the target, but to *attack* the one you've been given.
*   **The Mission:** You are not just "looking at" a program; you are on a **mission**. This framing is crucial. It implies a clear objective, a defined timeframe, and a commitment to see it through. The default 15-day duration is a meaningful but achievable sprint.
*   **The Reroll Friction:** The ability to reroll is intentionally difficult.
    *   **Limited Charges:** You only have 5 rerolls. This resource is precious, forcing you to question if you *really* need to switch or if you're just avoiding the hard work of deep investigation.
    *   **The Commitment Challenge:** The four-step confirmation process is a psychological barrier. It forces you to confront your own lack of discipline. Each "yes" is a small admission of defeat. More often than not, you will convince yourself to stick with the current mission by the third prompt.
    *   **The Lockout:** Exhausting your rerolls results in a lockout. This is the system's way of saying, "The time for choosing is over. The time for hunting is now." It removes all choice, leaving only the mission ahead.

## Installation

With Go (version 1.18+ or higher) installed, you can install `bounty-dice` with a single command:

```bash
go install github.com/xplo8e/bounty-dice@latest
```

This command will download the source code, compile it, and place the `bounty-dice` binary in your Go bin directory (usually `$HOME/go/bin`).

For the `bounty-dice` command to be accessible from anywhere, ensure this directory is in your system's `PATH`. You can check by running `echo $PATH`. If it's not there, add the following line to your shell's configuration file (e.g., `~/.zshrc`, `~/.bashrc`):

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

Then, restart your shell or run `source ~/.zshrc` to apply the changes.

## Configuration

The tool requires your HackerOne API credentials to fetch programs. These must be set as environment variables.

```bash
export HACKERONE_API_USER="your_api_username"
export HACKERONE_API_TOKEN="your_api_token"
```

You can add these lines to your shell profile (e.g., `~/.zshrc`, `~/.bashrc`) to make them permanent.

## Usage

The core command is simple:

```bash
./bounty-dice
```

This will roll the dice and assign you a new mission if you don't have an active one.

### Flags

*   `-bounty`: (Optional) Only include programs that offer bounties.
    ```bash
    ./bounty-dice -bounty
    ```
*   `-scope <type>`: (Optional) Filter by a specific asset type. Useful for focusing on your strengths.
    *   **Valid types:** `url`, `cidr`, `mobile`, `android`, `apple`, `ai`, `other`, `hardware`, `code`, `executable`
    ```bash
    ./bounty-dice -scope url -bounty
    ```
*   `-duration <days>`: (Optional) Set your mission duration. Must be between 15 and 30 days. Default is 15.
    ```bash
    ./bounty-dice -duration 30
    ```
*   `-reroll`: Abandon your current mission and start a new one. This will consume a reroll charge and trigger the Commitment Challenge.
    ```bash
    ./bounty-dice -reroll
    ```


