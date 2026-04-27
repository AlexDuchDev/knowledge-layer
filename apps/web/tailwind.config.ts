import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      maxWidth: {
        reading: "42rem",
        content: "72rem",
      },
    },
  },
  plugins: [],
};

export default config;
