package dev.gad.intellij.lang

import com.intellij.openapi.util.IconLoader

object GadIcons {
    /** The Gad logo (used for the run configuration and as a general mark). */
    @JvmField
    val LOGO = IconLoader.getIcon("/icons/gad.svg", GadIcons::class.java)

    /** File icons for the three dialects. */
    @JvmField
    val GAD_FILE = IconLoader.getIcon("/icons/gad-file.svg", GadIcons::class.java)

    @JvmField
    val GADT_FILE = IconLoader.getIcon("/icons/gadt-file.svg", GadIcons::class.java)

    @JvmField
    val GADX_FILE = IconLoader.getIcon("/icons/gadx-file.svg", GadIcons::class.java)

    /** `*_test.gad` files — the Gad file icon with a test badge. */
    @JvmField
    val GAD_TEST_FILE = IconLoader.getIcon("/icons/gad_test-file.svg", GadIcons::class.java)

    /** Backwards-compatible alias used by the run configuration. */
    @JvmField
    val FILE = LOGO
}
