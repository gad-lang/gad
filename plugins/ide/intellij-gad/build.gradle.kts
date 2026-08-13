import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("java")
    kotlin("jvm") version "2.0.21"
    id("org.jetbrains.intellij.platform") version "2.1.0"
}

group = providers.gradleProperty("pluginGroup").get()
version = providers.gradleProperty("pluginVersion").get()

repositories {
    mavenCentral()
    intellijPlatform {
        defaultRepositories()
        intellijDependencies()
    }
}

dependencies {
    intellijPlatform {
        create(
            providers.gradleProperty("platformType"),
            providers.gradleProperty("platformVersion"),
        )
        bundledPlugins(
            providers.gradleProperty("platformBundledPlugins").map { it.split(',').map(String::trim) },
        )

        pluginVerifier()
        zipSigner()
        instrumentationTools()
        testFramework(TestFrameworkType.Platform)
    }

    testImplementation("org.junit.jupiter:junit-jupiter:5.11.0")
}

kotlin {
    jvmToolchain(providers.gradleProperty("javaVersion").get().toInt())
}

intellijPlatform {
    // The plugin's settings are simple; skip the headless searchable-options build.
    buildSearchableOptions = false

    pluginConfiguration {
        id = providers.gradleProperty("pluginId")
        name = providers.gradleProperty("pluginName")
        version = providers.gradleProperty("pluginVersion")

        ideaVersion {
            sinceBuild = providers.gradleProperty("pluginSinceBuild")
            // No upper bound: the plugin uses only stable platform APIs, so it stays
            // compatible with current and future IDE builds (e.g. GoLand 2026.1+).
            untilBuild = provider { null }
        }
    }

    pluginVerification {
        ides {
            recommended()
        }
    }
}

// Assemble a TextMate bundle from the sibling VS Code extension (its package.json
// already declares the languages + grammars) and copy the config JSON schemas, so
// highlighting and schema validation share a single source of truth across the
// two editor plugins.
val vscodeDir = layout.projectDirectory.dir("../vscode-gad")

// A TextMate bundle is a VS Code-style directory: package.json + the grammars and
// language-configuration files it references. Ship the vscode-gad files verbatim.
val bundleGad by tasks.registering(Copy::class) {
    from(vscodeDir) {
        include("package.json")
        include("syntaxes/**")
        include("language-configuration.json")
        include("gadx-language-configuration.json")
    }
    into(layout.buildDirectory.dir("generated-resources/bundles/gad"))
}

val copySchemas by tasks.registering(Copy::class) {
    from(vscodeDir.dir("schemas")) { include("*.schema.json") }
    into(layout.buildDirectory.dir("generated-resources/schemas"))
}

sourceSets {
    main {
        resources.srcDir(layout.buildDirectory.dir("generated-resources"))
    }
}

tasks.named("processResources") {
    dependsOn(bundleGad, copySchemas)
}

tasks.test {
    useJUnitPlatform()
}
