---
title: UI Design Guide Applicability
doc_kind: engineering
doc_function: index
purpose: Records that project-level UI design references do not apply to the current CLI-only code-converge product.
derived_from:
  - ../../dna/governance.md
  - ../frontend.md
  - ../../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - project_ui_design_guide_applicability
must_not_define:
  - product_requirements
  - domain_rules
  - frontend_architecture_contract
  - feature_interface_requirements
  - implementation_source_of_truth
  - implementation_sequence
---

# UI Design Guide

N/A. `code-converge` has no graphical UI, UI kit, surface-specific components, screenshots, or frontend source paths. The current user interface is the CLI contract documented in the root [`README.md`](../../../README.md); frontend applicability is owned by [`../frontend.md`](../frontend.md).

No web, admin, mobile, or shared-component reference is kept in this project-specific Memory Bank. If a graphical surface is explicitly introduced later, replace this N/A record with references grounded in the implemented code and update [`../frontend.md`](../frontend.md). Do not copy generic surface templates into an active project document merely as placeholders.

- [Shared UI Components Guide](shared-components.md) — draft reference to adapt only after reusable UI assets exist.
