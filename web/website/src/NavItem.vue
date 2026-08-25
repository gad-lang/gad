<script setup lang="ts">
// A single sidebar navigation entry, rendered recursively: a leaf is a routed
// (or external) list item; an entry with `children` is an expandable submenu that
// renders its children with this same component, to arbitrary depth.
import { appBase, type NavPage } from "./content";

defineProps<{ item: NavPage }>();

const isExternal = (href: string) => /^https?:\/\//.test(href);
const navHref = (href: string) => (isExternal(href) ? href : appBase() + href);
</script>

<template>
  <v-list-group v-if="item.children && item.children.length" :value="'nav:' + item.title">
    <template #activator="{ props }">
      <v-list-item v-bind="props" :title="item.title" density="compact" />
    </template>
    <NavItem v-for="child in item.children" :key="child.slug || child.title" :item="child" />
  </v-list-group>

  <v-list-item
    v-else
    :title="item.title"
    :to="item.href ? undefined : { name: 'docs', params: { slug: item.slug } }"
    :href="item.href ? navHref(item.href) : undefined"
    :target="item.href && isExternal(item.href) ? '_blank' : undefined"
    :rel="item.href && isExternal(item.href) ? 'noopener' : undefined"
    :append-icon="item.href && isExternal(item.href) ? 'mdi-open-in-new' : undefined"
    color="primary"
    density="compact"
  />
</template>
