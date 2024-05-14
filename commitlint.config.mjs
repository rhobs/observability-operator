const Configuration = {
  /*
   * Resolve and load @commitlint/config-conventional from node_modules.
   */
  extends: ['@commitlint/config-conventional'],
  /*
   * Ignore dependabot commit messages until https://github.com/dependabot/dependabot-core/issues/2445 is fixed.
   */
  ignores: [(msg) => /Signed-off-by: dependabot\[bot]/m.test(msg)],
};

export default Configuration
