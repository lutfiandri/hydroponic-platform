# Agent instructions

- Prefer simple, stupid code over clever or over-engineered code.
- Never add backward compatibility unless I explicitly ask for it.
- For frontend shadcn/Tailwind UI changes, run `npm --prefix frontend run lint:shadcn`. It enforces `shadcn/no-restyle` and `shadcn/no-arbitrary-values` from `frontend/.oxlintrc.json`; do not restyle components or use arbitrary Tailwind values.
