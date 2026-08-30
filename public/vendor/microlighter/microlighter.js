const languageAliases = {
  docker: "dockerfile",
  gql: "graphql",
  js: "javascript",
  jsx: "javascript",
  md: "markdown",
  py: "python",
  rb: "ruby",
  sass: "scss",
  sh: "bash",
  shell: "bash",
  ts: "typescript",
  yml: "yaml",
  zsh: "bash"
};

/**
 * Convert a supported language alias to its grammar module name.
 * @param {string} language
 * @param {Object<string, string>} [aliases]
 * @returns {string}
 */
const normalizeLanguage = (language, aliases = {}) =>
  aliases[language] || languageAliases[language] || language;

/**
 * Find grammar modules referenced by external TextMate scope includes.
 * Local repository references (`#...`, `$self`, and `$base`) are ignored.
 * @param {*} value Grammar or nested grammar rule to inspect.
 * @returns {Set<string>} Normalized grammar module names.
 */
const getExternalLanguages = value => {
  const languages = new Set();

  const visit = item => {
    if (Array.isArray(item)) {
      item.forEach(visit);
    } else if (item && typeof item === "object") {
      if (typeof item.include === "string" && !/^[#$]/.test(item.include)) {
        const scope = item.include.split("#")[0];
        const match = scope.match(/^(?:source|text)\.([a-z0-9_-]+)/);
        if (match) languages.add(match[1]);
      }
      Object.values(item).forEach(visit);
    }
  };

  visit(value);
  return languages;
};

/**
 * Dynamically import a shipped grammar.
 * @param {string} language
 * @returns {Promise<Object | null>}
 */
const getGrammar = language => import(`./grammars/${language}.js`)
  .then(module => module.default)
  .catch(() => null);

/**
 * Create a cached loader that also resolves external grammar dependencies.
 * @param {(language: string) => Promise<Object | null>} [importLanguage]
 * @returns {(languages: string[]) => Promise<{
 *   languages: Object<string, Object>,
 *   scopes: Map<string, Object>
 * }>}
 */
const createGrammarLoader = (importLanguage = getGrammar) => {
  const loadingLanguages = new Map();
  const grammars = {
    languages: {},
    scopes: new Map()
  };

  /**
   * Import and index a grammar once for the lifetime of this loader.
   * @param {string} language
   * @returns {Promise<Object | null>}
   */
  const loadGrammar = language => {
    if (!loadingLanguages.has(language)) {
      const grammarModule = importLanguage(language).then(grammar => {
        if (grammar) {
          grammars.languages[language] = grammar;
          grammars.scopes.set(grammar.scopeName, grammar);
        }
        return grammar;
      });
      loadingLanguages.set(language, grammarModule);
    }

    return loadingLanguages.get(language);
  };

  return async languages => {
    const visited = new Set();
    let pending = [...new Set(languages)]
      .filter(language => /^[a-z0-9_-]+$/.test(language));

    while (pending.length) {
      pending.forEach(language => visited.add(language));
      const loaded = (await Promise.all(pending.map(loadGrammar))).filter(Boolean);
      pending = [...new Set(loaded.flatMap(grammar =>
        grammar.dependencies || [...getExternalLanguages(grammar)]
      ))].map(normalizeLanguage).filter(language => !visited.has(language));
    }

    return grammars;
  };
};

const getLanguageClass = element => [...element.classList]
  .find(className => className.startsWith("language-"))
  ?.slice("language-".length);

const getLanguage = codeBlock => {
  const pre = codeBlock.parentElement;
  const language = getLanguageClass(codeBlock)
    || codeBlock.dataset.language
    || getLanguageClass(pre)
    || pre.dataset.language
    // Deprecated: `lang` describes human language, not programming language.
    || pre.getAttribute("lang")
    || "";

  return language.toLowerCase();
};
const loadGrammars = createGrammarLoader();

/**
 * Highlight sets retained between scans so stale registered ranges can be
 * removed before replacements are created.
 * @type {Map<string, Highlight>}
 */
const highlights = new Map();

/**
 * Flatten a TextMate scope to a stable semantic CSS highlight category.
 * @param {string} scope
 * @returns {string | undefined}
 */
const getCategory = scope => {
  const parts = scope.split(".");
  const [first, second, third] = parts;
  const last = parts.at(-1);

  if (first === "markup" && ["quote", "inserted", "deleted", "raw"].includes(second)) return second;
  if (first === "entity" && second === "name") return third;
  if (scope.startsWith("constant.character.entity")) return "character-entity";
  if (parts.includes("numeric")) return "numeric";
  if (scope.startsWith("support.type.property-name")) return "property";
  if (parts.includes("attribute-value")) return "attribute-value";
  if (scope.startsWith("string.other.link")) return "link";

  if ([
    "doctype", "at-rule", "important", "regexp", "boolean",
    "symbol", "operator", "attribute-name"
  ].includes(last)) return last;

  if ([
    "comment", "string", "constant", "storage", "keyword",
    "variable", "punctuation", "entity", "support"
  ].includes(first)) return first;
};

/**
 * Register a matched text span under its CSS highlight category.
 * @param {Text} node
 * @param {number} start
 * @param {number} end
 * @param {string} scope
 * @returns {void}
 */
const addRange = (node, start, end, scope) => {
  const category = getCategory(scope);
  if (!category || start === end) return;

  const range = new Range();
  range.setStart(node, start);
  range.setEnd(node, end);

  if (!highlights.has(category)) highlights.set(category, new Highlight());
  highlights.get(category).add(range);
};

/**
 * Add named capture-group spans from a TextMate rule match.
 * @param {Text} node
 * @param {RegExpExecArray} match
 * @param {Object<string, {name: string}>} [captures]
 * @returns {void}
 */
const addCaptures = (node, match, captures = {}) => {
  Object.entries(captures).forEach(([index, capture]) => {
    const offsets = match.indices[index];
    if (offsets) addRange(node, ...offsets, capture.name);
  });
};

/**
 * Escape text before inserting it into a regular expression.
 * @param {string} value
 * @returns {string}
 */
const escapeRegex = value => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

/**
 * Replace TextMate end-pattern backreferences with escaped begin captures.
 * @param {string} pattern
 * @param {RegExpExecArray} beginMatch
 * @returns {string}
 */
const expandEnd = (pattern, beginMatch) => pattern.replace(/\\(\d+)/g, (reference, index) => {
  return beginMatch[index] === undefined ? reference : escapeRegex(beginMatch[index]);
});

/**
 * Normalize a grammar rule or repository entry to a rule array.
 * @param {*} rule
 * @returns {Object[]}
 */
const getRules = rule => {
  if (!rule) return [];
  if (Array.isArray(rule)) return rule;
  if (rule.match || rule.begin || rule.include) return [rule];
  return rule.patterns || [];
};

/**
 * Find code blocks, load their TextMate grammars, and register CSS highlights.
 * @param {Object} [options]
 * @param {ParentNode} [options.root=document]
 * @param {string} [options.selector="pre > code"]
 * @param {Object<string, string>} [options.languageAliases]
 * @returns {Promise<HTMLElement[]>}
 */
const highlightAll = async ({
  root = document,
  selector = "pre > code",
  languageAliases
} = {}) => {
  const codeBlocks = [...root.querySelectorAll(selector)].filter(getLanguage);
  const languages = codeBlocks.map(codeBlock =>
    normalizeLanguage(getLanguage(codeBlock), languageAliases)
  );
  const grammars = await loadGrammars(languages);
  const regexes = new Map();

  highlights.forEach((ranges, category) => {
    if (CSS.highlights.get(category) === ranges) CSS.highlights.delete(category);
    ranges.clear();
  });

  /**
   * Find the next bounded match while reusing compiled grammar expressions.
   * @param {string} pattern
   * @param {Text} node
   * @param {number} start
   * @param {number} end
   * @returns {RegExpExecArray | null}
   */
  const exec = (pattern, node, start, end) => {
    if (!regexes.has(pattern)) regexes.set(pattern, new RegExp(pattern, "dgm"));
    const regex = regexes.get(pattern);
    regex.lastIndex = start;
    const match = regex.exec(node.data);
    return match && match.indices[0][0] < end && match.indices[0][1] <= end ? match : null;
  };

  /**
   * Resolve local, base, and external TextMate include references.
   * @param {string} include
   * @param {Object} grammar
   * @param {Object} baseGrammar
   * @returns {{grammar: Object, rules: Object[]} | null}
   */
  const resolveInclude = (include, grammar, baseGrammar) => {
    if (include === "$self") return { grammar, rules: grammar.patterns };
    if (include === "$base") return { grammar: baseGrammar, rules: baseGrammar.patterns };

    if (include[0] === "#") {
      return { grammar, rules: getRules(grammar.repository?.[include.slice(1)]) };
    }

    const [scopeName, repositoryName] = include.split("#");
    const includedGrammar = grammars.scopes.get(scopeName);
    if (!includedGrammar) return null;

    return {
      grammar: includedGrammar,
      rules: repositoryName
        ? getRules(includedGrammar.repository?.[repositoryName])
        : includedGrammar.patterns
    };
  };

  /**
   * Expand include rules into directly matchable rule contexts.
   * @param {Object[]} rules
   * @param {Object} grammar
   * @param {Object} baseGrammar
   * @param {Set<string>} [activeIncludes]
   * @returns {{grammar: Object, rule: Object}[]}
   */
  const expandRules = (rules, grammar, baseGrammar, activeIncludes = new Set()) => {
    const expanded = [];

    rules.forEach(rule => {
      if (rule.include) {
        const includeKey = `${grammar.scopeName}:${rule.include}`;
        if (activeIncludes.has(includeKey)) return;

        const included = resolveInclude(rule.include, grammar, baseGrammar);
        if (!included) return;

        const nestedIncludes = new Set(activeIncludes);
        nestedIncludes.add(includeKey);
        expanded.push(...expandRules(included.rules, included.grammar, baseGrammar, nestedIncludes));
        return;
      }

      if (rule.match || (rule.begin && rule.end)) expanded.push({ grammar, rule });
    });

    return expanded;
  };

  /**
   * Select the rule with the earliest match in a text region.
   * @param {Text} node
   * @param {{grammar: Object, rule: Object}[]} contexts
   * @param {number} start
   * @param {number} end
   * @returns {{grammar: Object, rule: Object, match: RegExpExecArray} | null}
   */
  const nextRule = (node, contexts, start, end) => {
    let winner = null;

    contexts.forEach(context => {
      const pattern = context.rule.match || context.rule.begin;
      const match = exec(pattern, node, start, end);
      if (!match) return;

      if (!winner || match.indices[0][0] < winner.match.indices[0][0]) {
        winner = { ...context, match };
      }
    });

    return winner;
  };

  /**
   * Scan a bounded text region, recursively handling begin/end rule pairs.
   * @param {Text} node
   * @param {Object[]} rules
   * @param {number} start
   * @param {number} end
   * @param {Object} grammar
   * @param {Object} [baseGrammar]
   * @param {{pattern: string, applyEndPatternLast?: boolean} | null} [closing]
   * @returns {{contentEnd: number, end: number, match: RegExpExecArray | null}}
   */
  const scanRegion = (node, rules, start, end, grammar, baseGrammar = grammar, closing = null) => {
    const contexts = expandRules(rules, grammar, baseGrammar);
    let cursor = start;

    while (cursor < end) {
      const candidate = nextRule(node, contexts, cursor, end);
      const endMatch = closing ? exec(closing.pattern, node, cursor, end) : null;
      const candidateStart = candidate?.match.indices[0][0] ?? Infinity;
      const endStart = endMatch?.indices[0][0] ?? Infinity;

      if (endMatch && (endStart < candidateStart || (endStart === candidateStart && !closing.applyEndPatternLast))) {
        return { contentEnd: endStart, end: endMatch.indices[0][1], match: endMatch };
      }

      if (!candidate) return { contentEnd: end, end, match: null };

      const { rule, match, grammar: ruleGrammar } = candidate;
      if (rule.match) {
        if (rule.name) addRange(node, ...match.indices[0], rule.name);
        addCaptures(node, match, rule.captures);
        cursor = match.indices[0][1] > cursor ? match.indices[0][1] : cursor + 1;
        continue;
      }

      const nested = scanRegion(
        node,
        rule.patterns || [],
        match.indices[0][1],
        end,
        ruleGrammar,
        baseGrammar,
        { pattern: expandEnd(rule.end, match), applyEndPatternLast: rule.applyEndPatternLast }
      );

      if (rule.name) addRange(node, match.indices[0][0], nested.end, rule.name);
      if (rule.contentName) addRange(node, match.indices[0][1], nested.contentEnd, rule.contentName);
      addCaptures(node, match, rule.beginCaptures || rule.captures);
      if (nested.match) addCaptures(node, nested.match, rule.endCaptures || rule.captures);

      cursor = nested.end > cursor ? nested.end : cursor + 1;
    }

    return { contentEnd: end, end, match: null };
  };

  codeBlocks.forEach((codeBlock, index) => {
    const grammar = grammars.languages[languages[index]];
    if (!grammar) return;

    codeBlock.normalize();
    const node = codeBlock.firstChild;
    if (node?.nodeType !== Node.TEXT_NODE || node.nextSibling) return;

    scanRegion(node, grammar.patterns, 0, node.data.length, grammar);
  });

  highlights.forEach((ranges, category) => {
    if (ranges.size) CSS.highlights.set(category, ranges);
  });

  return codeBlocks;
};

document.addEventListener("syntax-highlight", () => highlightAll());
highlightAll();
