# Agent instructions

- Prefer simple, stupid code over clever or over-engineered code.
- Never add backward compatibility unless I explicitly ask for it.
- Make small, focused commits for each coherent change. Do not bundle unrelated changes into a bulk commit.
- Always use antislop in `during` mode for planning and UI implementation. This is the user's standing preference; never ask which antislop mode to use again.
- For frontend shadcn/Tailwind UI changes, run `npm --prefix frontend run lint:shadcn`. It enforces `shadcn/no-restyle` and `shadcn/no-arbitrary-values` from `frontend/.oxlintrc.json`; do not restyle components or use arbitrary Tailwind values.
