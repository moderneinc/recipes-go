@file:Suppress("GradlePackageUpdate", "UnstableApiUsage")

import com.github.jengelman.gradle.plugins.shadow.tasks.ShadowJar
import com.hierynomus.gradle.license.tasks.LicenseCheck
import com.hierynomus.gradle.license.tasks.LicenseFormat

plugins {
    id("org.openrewrite.build.recipe-library") version "latest.release"
    id("org.openrewrite.build.moderne-proprietary-license") version "latest.release"
    id("com.gradleup.shadow") version "8.3.0"
}

group = "org.openrewrite.recipe"
description = "Go code quality, migration, and remediation recipes."

// Disable javadoc when there are no public Java classes
tasks.withType<Javadoc> {
    isEnabled = false
}

// No Java tests in this project
tasks.withType<Test> {
    isEnabled = false
}

val rewriteVersion = rewriteRecipe.rewriteVersion.get()
dependencies {
    implementation(platform("org.openrewrite:rewrite-bom:${rewriteVersion}"))
    implementation("org.openrewrite:rewrite-java")
    implementation("org.openrewrite:rewrite-go:${rewriteVersion}")
    implementation("io.moderne:jsonrpc:latest.integration")
}

// ============================================
// License headers on Go files
// ============================================
// Java/Kotlin/TS are handled by the recipe-library convention plugin; here we extend
// the same hierynomus license plugin to also check/format .go files.

val goLicenseSources = fileTree(projectDir) {
    include("**/*.go")
    exclude("**/build/**", "**/vendor/**")
}

val licenseGo by tasks.registering(LicenseCheck::class) {
    source = goLicenseSources
    header = rootProject.file("gradle/licenseHeader.txt")
    mapping("go", "SLASHSTAR_STYLE")
}

val licenseFormatGo by tasks.registering(LicenseFormat::class) {
    source = goLicenseSources
    header = rootProject.file("gradle/licenseHeader.txt")
    mapping("go", "SLASHSTAR_STYLE")
}

tasks.named("license") { dependsOn(licenseGo) }
tasks.named("licenseFormat") { dependsOn(licenseFormatGo) }

// ============================================
// Java RPC Test Server (shadow JAR for Go tests that need Java delegation)
// ============================================

val rpcTestServer by tasks.registering(ShadowJar::class) {
    group = "go"
    description = "Build the Java RPC test server fat JAR for Go integration tests"

    archiveClassifier.set("rpc-test-server")
    from(sourceSets["main"].output)
    configurations = listOf(project.configurations.runtimeClasspath.get())
    manifest {
        attributes("Main-Class" to "org.openrewrite.maven.rpc.JavaRewriteRpc")
    }
    mergeServiceFiles()
}

// ============================================
// Go Build Tasks
// ============================================

// Find go executable
fun findGo(): String {
    val candidates = listOf("go")
    for (cmd in candidates) {
        try {
            val process = ProcessBuilder(cmd, "version")
                .redirectErrorStream(true)
                .start()
            if (process.waitFor() == 0) {
                return cmd
            }
        } catch (e: Exception) {
            // Command not found, try next
        }
    }
    throw GradleException("Go toolchain not found. Please install Go 1.23+ and ensure 'go' is on your PATH.")
}

// Register build/test tasks for a Go module directory
fun registerGoModule(taskPrefix: String, dirName: String, needsRpcServer: Boolean = false) {
    val moduleDir = projectDir.resolve(dirName)

    val modDownload = tasks.register("${taskPrefix}Restore", Exec::class.java) {
        group = "go"
        description = "Download Go module dependencies for $dirName"

        onlyIf { moduleDir.exists() }

        workingDir = moduleDir
        commandLine(findGo(), "mod", "download")

        doFirst {
            logger.lifecycle("Downloading Go module dependencies in $moduleDir")
        }
    }

    val build = tasks.register("${taskPrefix}Build", Exec::class.java) {
        group = "go"
        description = "Build Go packages in $dirName"

        dependsOn(modDownload)
        onlyIf { moduleDir.exists() }

        workingDir = moduleDir
        commandLine(findGo(), "build", "./...")

        doFirst {
            logger.lifecycle("Building Go packages in $moduleDir")
        }
    }

    val test = tasks.register("${taskPrefix}Test", Exec::class.java) {
        group = "go"
        description = "Run Go tests in $dirName"

        dependsOn(build)
        if (needsRpcServer) {
            dependsOn(rpcTestServer)
        }
        onlyIf { moduleDir.exists() }

        workingDir = moduleDir
        if (needsRpcServer) {
            environment("RPC_TEST_SERVER_JAR", rpcTestServer.get().archiveFile.get().asFile.absolutePath)
        }
        commandLine(findGo(), "test", "./...", "-count=1", "-v")

        doFirst {
            logger.lifecycle("Running Go tests in $moduleDir")
            if (needsRpcServer) {
                logger.lifecycle("RPC test server JAR: ${rpcTestServer.get().archiveFile.get().asFile.absolutePath}")
            }
        }
    }

    // Integrate into the check task
    tasks.named("check") {
        if (moduleDir.exists()) {
            dependsOn(test)
        }
    }

    // Make the conventional `./gradlew test` also run Go tests, even though the
    // JVM Test task itself is disabled (see tasks.withType<Test> above).
    tasks.named("test") {
        if (moduleDir.exists()) {
            dependsOn(test)
        }
    }
}

// ============================================
// Module registrations
// ============================================

registerGoModule("codeQuality", "recipes-code-quality", needsRpcServer = false)
