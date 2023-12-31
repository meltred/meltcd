/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        rootBg: "#14171c",
        sidebar: "#22272E",
        sidebarLite: "#363e49",
      },
    },
  },
  plugins: [],
};
