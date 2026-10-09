# strata — Exploración profunda

**Fecha:** 2026-10-09
**Estado:** idea. Sin código.
**Decisiones tomadas (2026-10-09):** ecosistema de dos tools — `spec-blame`
(verifica spec→código) + `spec-excavate` (mina código→spec) — monorepo Go
con `pkg/reqgraph` compartido. Umbrella: **strata** (temática geología:
excavate mina capas de código, blame las anota).
**TL;DR:** el problema (specs y código divergen, no existe `coverage` para
requirements) es real y está *caliente* — pero ya no está vacío. En los
últimos meses aparecieron 4+ herramientas nuevas haciendo "spec coverage +
drift gate". Todas son pre-1.0, sin ganador, fragmentadas en formatos
propietarios de spec. El hueco defendible que queda: **ingesta
format-agnostic (openspec-first) + blame a nivel símbolo + binario Go**.
Un clon más del coverage gate no vale la pena.

---

## 1. El problema

Spec-driven development está en su momento (GitHub Spec Kit, Kiro, openspec,
Tessl, Claude Code skills). Todos esos flujos tratan la spec como *input* de
generación de código. Ninguno responde, tres semanas después: "¿el código
todavía implementa lo que dice la spec? ¿y esta función, qué requirement la
justifica?"

Las herramientas clásicas de requirements traceability (DOORS, Jama,
Polarion) cuestan miles de USD/asiento y están pensadas para auditorías de
aviónica/medicina, no para el loop dev. El hueco conceptual que nadie
empaquetó bien: **requirements traceability como dev tool, no como
compliance tool** — un `git blame` donde cada línea responde "qué
requirement me justifica", y un CI gate donde la spec es la que aprueba.

## 2. Panorama competitivo (verificado 2026-10)

### Guardia vieja — orientada a compliance/auditoría

| Proyecto | Stack | Qué hace | Problema para nuestro caso |
|---|---|---|---|
| **StrictDoc** | Python, Apache-2.0 | Gestión completa: SDoc format, grafo de traceability, *source coverage* con parsing language-aware (C/C++/Python), matriz HTML, ReqIF | Te obliga a adoptar SDoc; UX web-pesada; pensado para DO-178C |
| **OpenFastTrace** | Java/JAR, Maven/Gradle | Req tracing suite clásico, coverage report | Java-céntrico, DX de 2015 |
| **Doorstop** | Python | Requirements como YAML per-item en git, links entre docs | Gestión, no blame; formato propio |
| **reqcov** | Python (PyPI) | "Codecov para requirements": markers `@req`/`@implements`/`@verifies` en cualquier lenguaje, merge JUnit → verified/failing, gate configurable | Muy cercano funcionalmente; file/marker-level, sin símbolos |
| **pytreqt** | Python | pytest plugin, req IDs en docstrings | Solo pytest, narrow |

### Oleada nueva — spec-driven dev era (todas pre-1.0, 2025-2026)

| Proyecto | Stack | Pitch | Señal de mercado |
|---|---|---|---|
| **spec-seal** (xantus-ai) | TypeScript/npm | Spec coverage + drift. Specs MD con IDs, código referencia con `@spec REQ-1 #a3f2b1` — **hash-bound**: hash corto de los acceptance criteria al momento de anotar; si la spec cambia, la anotación queda *stale*. `check/coverage/map/sync` | 1 star, 7 commits. El truco del hash es *más elegante* que detectar drift por diff |
| **@spec-engine/spec-engine** | TypeScript/npm | Cross-repo: requirements con IDs permanentes, tags `@spec`, índice sqlite derivado, `spec check --ci`, `spec guard` (bloquea borrar req), `spec serve` webapp, **MCP server incluido** | Más completo; npm package, privado-ish |
| **spec-trace** | Python (PyPI) | Lint spec↔tests: bullets MD con keycodes ↔ tests que citan keycodes. Zero deps, CI gate. Insight bueno: el gate solo falla por refs rotas/typos, coverage es *advisory* — "un gate que falla por coverage se borra al mes" | Reporta 167 spec files / 3.6k spec points en uso real |
| **spec-blame (este)** | — | — | — |

### Lectura honesta

- **El core idea ya está inventado, varias veces y muy recientemente.** "Coverage
  para requirements + CI gate" es literalmente el pitch de spec-seal, reqcov y
  spec-trace. spec-engine ya incluye MCP y webapp.
- Pero: **ninguno ganó.** Todos pre-1.0, todos con formato de spec propio,
  tracción ~nula (spec-seal: 1 star). Es una carrera que recién arranca —
  entrar no es llegar tarde, es llegar a la grilla.
- El riesgo real: es un espacio donde un incumbent (GitHub, Tessl, openspec
  mismo) puede absorber la feature. La defensa es ser **format-agnostic**:
  los demás te venden *su* spec format; nosotros indexamos la que ya tenés.

## 3. El hueco defendible

Tres vectores donde ninguno de los existentes es fuerte:

1. **Ingesta format-agnostic, openspec-first.** spec-seal/spec-engine definen
   formatos propios (¡otro formato más!). spec-blame parsea las specs que
   *ya existen*: openspec `spec.md` con `#### Scenario:` Given/When/Then,
   spec-kit, Gherkin `.feature`, markdown plano con IDs, y hasta GitHub
   issues. Adapter por formato → grafo canónico de requirements.
   **Esto es el wedge**: te sumás al flujo SDD que el usuario ya eligió.
2. **Blame a nivel símbolo.** Los nuevos son file/marker-level; StrictDoc
   parsea C/C++/Python. La versión piola: índice de símbolos vía SCIP
   (sourcegraph) o tree-sitter → `spec-blame foo.py` anota *por función*.
   Habilita dead-code radar real: "este símbolo no responde a ningún req".
3. **Superficie para agentes + humanos.** CLI + CI gate para humanos, MCP
   server (`req_for_symbol`, `symbols_for_req`, `coverage`, `stale`) para
   que sdd-apply/Claude/Devin consulten el mapa *mientras* codean. Nadie
   combina las dos bien; spec-engine tiene MCP pero sin symbol-level.

## 4. Mecánica de mapeo requirement→código

En orden de confianza (todas coexisten):

| Mecanismo | Cómo | Precisión |
|---|---|---|
| **Marker explícito** | `# spec: auth/REQ-04` o `@implements AUTH-04` en comentario/docstring. Determinístico, auditable | Total — pero requiere disciplina |
| **Hash-bound staleness** (robar de spec-seal) | marker lleva hash corto del acceptance criteria; spec cambia → stale flag. Staleness *semántico* sin LLM | Total |
| **Convención BDD/test** | pytest-bdd/behave/cucumber linkean scenario→step def gratis; nombres de test que citan IDs | Alta |
| **SCIP index** | indexadores por lenguaje (scip-go, scip-python...) → símbolos con ubicación exacta | Alta, sin escribir parsers |
| **Inferencia LLM** | sugerir mappings scenario↔símbolo no anotados. *Sugerencia*, nunca gate | Media — solo para bootstrap |

Insight de spec-trace a internalizar: **el gate solo debe fallar por cosas
objetivamente rotas** (refs a reqs inexistentes, markers stale, req sin
ningún link). Coverage bajo es advisory o se configura — si el gate es
ruidoso, lo desinstalan.

## 5. Arquitectura propuesta (Go)

```
spec-blame
├── ingest/    adapters: openspec | spec-kit | gherkin | md-ids | gh-issues
│              → RequirementGraph canónico (id, text, hash, parent, source)
├── index/     símbolos: SCIP protobuf (preferido) o tree-sitter wasm
│              (wazero → sin CGO → cross-compile limpio)
├── markers/   scanner de anotaciones (regex por comentario de cada
│              lenguaje — no necesita parser, es grep con estructura)
├── graph/     join RequirementGraph × Symbols × Markers × Tests
├── report/    blame (annotate), coverage, stale, orphans, dead-code
├── gate/      check --ci: exit codes + SARIF output (GitHub code scanning!)
├── serve/     MCP server (mismo grafo, surface para agentes)
└── tui/       bubbletea: navegar req↔símbolo, fuzz por spec (post-MVP)
```

## 6. Decisión de lenguaje — Go, honestamente

La velocidad no es el cuello de botella (parsear un repo es I/O + un
tree-sitter que es C igual en todos los lenguajes). Las ventajas reales de
Go acá son **distribución y UX de CI**:

- Binario estático único → `spec-blame check` corre en cualquier CI sin
  runtime (los competidores: `npm install -g` + node, o pip + venv).
- Startup ~0ms — en CI gate importa que se sienta gratis.
- Bubbletea para el TUI (ya lo conocés).
- `goldmark` para parsear specs MD, `sourcegraph/scip` para índices.
- Único asterisco: tree-sitter en Go es CGO (`tree-sitter/go-tree-sitter`).
  Alternativa pure-Go: gramáticas compiladas a WASM + wazero — sin CGO,
  cross-compile trivial. O no usar tree-sitter en MVP: markers son regex,
  símbolos salen de SCIP.

Contra Go: si la mitad del valor termina siendo "inferencia semántica con
LLM", Python integra mejor. Por eso el orden importa: MVP = determinístico
(markers + hash-bound + SCIP), LLM queda como capa opcional externa
(consumir cualquier endpoint OpenAI-compatible).

## 7. MVP

1. `spec-blame scan` — parsear openspec/spec-kit + scan markers → grafo
2. `spec-blame check` — refs rotas, stale (hash-bound), reqs sin link → exit code
3. `spec-blame blame <file>` — anotación por línea/símbolo
4. `spec-blame coverage` — % + tabla
5. GitHub Action (como microburst: `uses: pingedbrain/spec-blame@v1`)

Post-MVP: SCIP symbol-level real, MCP server, TUI, dead-code radar,
LLM-suggest.

## 8. Riesgos

- **Espacio en carrera**: 4+ entradas en meses, puede aparecer un winner o
  GitHub lo absorbe → defensa = format-agnostic + symbol-level + Go DX.
- **Format-agnostic = n parsers**: cada adapter es trabajo; mitigar con un
  IR mínimo y community adapters.
- **Markers requieren disciplina**: si nadie anota, coverage=0 y el tool se
  ve vacío → mitigar con `init` que bootstrappea markers vía LLM-suggest
  (feature, no dependencia).
- **Nombre**: `spec-blame` parece libre; verificar npm/pypi/crates antes.

## 9. Veredicto

**Go condicional → aceptado como ecosistema.** Vale la pena si el eje es
*"el traceability tool que entra en tu flujo SDD existente y baja al
símbolo"* — no si es "otro coverage gate" (esa guerra ya está picando y
con JS primero).

**Decisión: los dos como ecosistema.** spec-blame solo es contested;
spec-excavate solo tiene un wedge claro pero poco hook. Juntos cubren el
ciclo completo de specs en brownfield: excavate escribe la spec que el
repo "debería haber tenido", blame la mantiene verdadera.

```
strata/                        # monorepo Go, umbrella
├── pkg/reqgraph               # IR canónico: Requirement, Scenario, hash, parent
├── pkg/ingest                 # openspec | spec-kit | gherkin | md-ids (lectura)
├── pkg/emit                   # spec.md openspec-compatible (escritura)
├── pkg/index                  # símbolos: SCIP / tree-sitter-wasm (wazero, sin CGO)
├── pkg/markers                # @spec/@implements scanner + staleness (hash-bound)
├── cmd/spec-blame             # verify: check | blame | coverage | map | serve(MCP)
├── cmd/spec-excavate          # mine: scan → propone spec.md + markers sugeridos
└── action.yml                 # GitHub Action (patrón microburst)
```

Flujo vendible: `spec-excavate` genera `openspec/specs/` en repo legacy →
`spec-blame check` lo gatea en CI → `serve` expone el grafo a agentes
(sdd-apply, Claude, Devin) mientras codean. "SDD para brownfield"
end-to-end — nadie lo empaquetó.

Nombre `strata` verificado (2026-10-09): `pingedbrain/strata` libre,
binarios `spec-blame`/`spec-excavate` libres en npm+PyPI. `strata` como
tal está tomado en npm/PyPI y existe una org GitHub chica (headless CMS,
13★) + OpenGamma Strata (Java, finanzas) — colisión baja porque el
umbrella es solo el repo; los artefactos distribuidos son los binarios.
