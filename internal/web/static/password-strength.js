// Adds a strength meter under each password field marked with
// data-password-strength. The estimate comes from zxcvbn-ts
// (https://zxcvbn-ts.github.io/zxcvbn/, MIT license), whose browser builds the
// page loads before this script. The meter is advice only: the server enforces
// the password rules. Without JavaScript nothing is shown.

const { core, "language-common": common, "language-en": en } = window.zxcvbnts;
const zxcvbn = new core.ZxcvbnFactory({
  dictionary: { ...common.dictionary, ...en.dictionary },
  graphs: common.adjacencyGraphs,
  translations: en.translations,
});

// zxcvbn's score from 0 to 4, in the words of its documentation.
const scoreLabels = ["Too guessable", "Very guessable", "Somewhat guessable", "Safely unguessable", "Very unguessable"];

for (const input of document.querySelectorAll("input[type=password][data-password-strength]")) {
  const strength = document.createElement("div");
  strength.dataset.passwordStrength = "";
  strength.hidden = true;

  // Scores 0 and 1 show as poor, 2 as fair, and 3 and 4 as good.
  const meter = document.createElement("meter");
  meter.min = 0;
  meter.max = 4;
  meter.low = 2;
  meter.high = 3;
  meter.optimum = 4;
  // The text below says the same, so screen readers skip the bar.
  meter.setAttribute("aria-hidden", "true");

  const text = document.createElement("small");
  text.setAttribute("aria-live", "polite");

  strength.append(meter, text);
  // Below the field's hint or error.
  const description = document.getElementById(input.getAttribute("aria-describedby"));
  (description ?? input).after(strength);

  // The other fields of the form, such as the email and the names, count as
  // easy to guess when they appear in the password.
  const userInputs = () =>
    [...input.form.querySelectorAll("input[type=email], input[type=text]")]
      .filter((field) => field !== input && field.value !== "")
      .map((field) => field.value);

  const update = () => {
    strength.hidden = input.value === "";
    if (strength.hidden) {
      return;
    }
    const result = zxcvbn.check(input.value, userInputs());
    meter.value = result.score;
    text.textContent = [`${scoreLabels[result.score]}.`, result.feedback.warning, ...result.feedback.suggestions]
      .filter(Boolean)
      .join(" ");
  };
  input.form.addEventListener("input", update);
}
