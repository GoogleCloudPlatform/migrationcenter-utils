# **CodMod App Modernization Assessment:**   **Report Options**

This document provides a brief overview of the different types of assessment reports that can be generated using the CodMod tool.

### **1\. Standard Default Report ([Sample report](https://github.com/GoogleCloudPlatform/migrationcenter-utils/blob/f829a6bdccaf9ebe20e17f5efabf3318ff913633/tools/codmod/sample-reports/shopping-cart.html))**

This report provides a high-level, less detailed assessment of the codebase. It's ideal for quick insights, executive summaries, and early-stage reviews when a comprehensive deep-dive is not required.

### **2\. Standard Full Report ([Sample report](https://github.com/GoogleCloudPlatform/migrationcenter-utils/blob/f829a6bdccaf9ebe20e17f5efabf3318ff913633/tools/codmod/sample-reports/shopping-cart.html))**

This is a more comprehensive assessment designed to accelerate early-stage planning. It analyzes the codebase to identify key components, flag potential blockers or security risks, and provides a solid baseline for modernization.

### **3\. Intent-Based Report ([Sample report](https://github.com/GoogleCloudPlatform/migrationcenter-utils/blob/main/tools/codmod/sample-reports/cymbal-coffee-microsoft-modernization.html))**

This report is tailored to a specific modernization goal, such as migrating from a legacy Java version or another cloud provider. It provides a focused roadmap and analysis relevant to the specified "intent" (e.g., JAVA\_LEGACY\_TO\_MODERN).

### **4\. Data Layer Report ([Sample report](https://github.com/GoogleCloudPlatform/migrationcenter-utils/blob/main/tools/codmod/sample-reports/spring-petclinic-data-layer.html))**

This report focuses specifically on the database-related aspects of the application. It provides a structured view of the data architecture, identifies schemas and dependencies, and recommends modernization opportunities for Google Cloud database services.

### **5\. Report with Optional Sections (Files/Classes) ([Sample report](https://github.com/GoogleCloudPlatform/migrationcenter-utils/blob/main/tools/codmod/sample-reports/spring-petclinic-optional-sections.html))**

This option enhances a standard report by adding detailed sections on the codebase's file structure and class dependencies. It's useful when you need a deeper understanding of the code's organization and interconnections.

### **6\. Custom & Revised Reports ([Sample report](http://spring-petclinic-optional-sections.html))**

These options offer maximum flexibility by allowing you to create or modify reports based on your own context and questions. You can generate new sections, revise existing ones with new information, and tailor the output to meet specific project requirements or address stakeholder feedback.