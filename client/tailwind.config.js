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
        border: "rgba(255, 255, 255, 0.08)",
        input: "rgba(255, 255, 255, 0.05)",
        ring: "rgba(139, 157, 255, 0.4)",
        background: "#0E1116",
        foreground: "#FFFFFF",
        primary: {
          DEFAULT: "#8B9DFF",
          foreground: "#FFFFFF",
        },
        secondary: {
          DEFAULT: "#131720",
          foreground: "#9AA4B2",
        },
        accent: {
          DEFAULT: "#A8FFDA",
          foreground: "#0E1116",
        },
        muted: {
          DEFAULT: "rgba(255, 255, 255, 0.05)",
          foreground: "#9AA4B2",
        },
        card: {
          DEFAULT: "rgba(255, 255, 255, 0.05)",
          foreground: "#FFFFFF",
        },
        popover: {
          DEFAULT: "rgba(255, 255, 255, 0.08)",
          foreground: "#FFFFFF",
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
