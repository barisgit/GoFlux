// Test JavaScript file for static serving
console.log("Hello from test app.js!");

function testFunction() {
  console.log("Test function called");
  return "test-result";
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = { testFunction };
}
