/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
  ],
  // Progress-bar widths (today-programs.html, schedule.js) are set by toggling
  // one of these classes instead of an inline `style="width:...` attribute, so
  // they render under a CSP with no `style-src 'unsafe-inline'`. Progress is
  // always an integer 0-100 (see models.Progress), so 0%-100% covers every value.
  safelist: Array.from({ length: 101 }, (_, i) => `w-[${i}%]`),
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: "#031c4f", // ClassyFM navy (sourced from classyfm.co.id's own CSS)
          dark: "#010f30",
          light: "#26468f",
        },
        ink: "#141414",
      },
      fontFamily: {
        sans: ["Manrope", "system-ui", "sans-serif"],
        heading: ["Poppins", "system-ui", "sans-serif"],
      },
    },
  },
  plugins: [],
};
