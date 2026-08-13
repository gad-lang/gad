package dev.gad.intellij.config

import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.jetbrains.jsonSchema.extension.JsonSchemaFileProvider
import com.jetbrains.jsonSchema.extension.JsonSchemaProviderFactory
import com.jetbrains.jsonSchema.extension.SchemaType

/**
 * Validates and completes the Gad project config files against the JSON schemas
 * reused from the VS Code extension:
 *   .gad.yaml / .gad.yml     → gad-config.schema.json
 *   .gadide.yaml / .gadide.yml → gadide-config.schema.json
 */
class GadJsonSchemaProviderFactory : JsonSchemaProviderFactory {
    override fun getProviders(project: Project): List<JsonSchemaFileProvider> = listOf(
        SchemaProvider("Gad config", "/schemas/gad-config.schema.json", setOf(".gad.yaml", ".gad.yml")),
        SchemaProvider("Gad IDE config", "/schemas/gadide-config.schema.json", setOf(".gadide.yaml", ".gadide.yml")),
    )

    private class SchemaProvider(
        private val presentableName: String,
        private val resource: String,
        private val fileNames: Set<String>,
    ) : JsonSchemaFileProvider {

        override fun isAvailable(file: VirtualFile): Boolean = file.name in fileNames

        override fun getName(): String = presentableName

        override fun getSchemaFile(): VirtualFile? =
            JsonSchemaProviderFactory.getResourceFile(GadJsonSchemaProviderFactory::class.java, resource)

        override fun getSchemaType(): SchemaType = SchemaType.embeddedSchema
    }
}
