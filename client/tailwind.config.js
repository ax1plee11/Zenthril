/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ["class"],
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        border: "var(--border)",
        input: "var(--bg-input)",
        ring: "var(--border-accent)",
        background: "var(--bg-base)",
        foreground: "var(--text-primary)",
        primary: {
          DEFAULT: "var(--accent)",
          foreground: "var(--on-accent)",
        },
        secondary: {
          DEFAULT: "var(--bg-elevated)",
          foreground: "var(--text-secondary)",
        },
        accent: {
          DEFAULT: "var(--accent-dim)",
          foreground: "var(--bg-base)",
        },
        muted: {
          DEFAULT: "var(--bg-input)",
          foreground: "var(--text-secondary)",
        },
        card: {
          DEFAULT: "var(--bg-input)",
          foreground: "var(--text-primary)",
        },
        popover: {
          DEFAULT: "var(--border)",
          foreground: "var(--text-primary)",
        },
      },
      borderRadius: {
        lg: "16px",
        md: "12px",
        sm: "10px",
      },
      backdropBlur: {
        glass: "16px",
      },
      boxShadow: {
        glass: "0 10px 30px rgba(0, 0, 0, 0.35)",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
}
