// Permissive re-exports of the Vuetify components used by the TSX views.
//
// Vuetify's generated component types do not expose native/fallthrough
// attributes (`onClick`, `title`, `onKeyup`, …), which templates accept freely
// but TSX rejects. Re-exporting each component as a loosely-typed component lets
// the JSX accept those attributes. Our own code (controller, IdeApi, props and
// emits) stays fully typed; only Vuetify's internal prop-checking is relaxed.
import type { DefineComponent } from "vue";
import {
  VBtn as _VBtn,
  VBtnToggle as _VBtnToggle,
  VCard as _VCard,
  VCardActions as _VCardActions,
  VCardText as _VCardText,
  VCardTitle as _VCardTitle,
  VCheckbox as _VCheckbox,
  VChip as _VChip,
  VCombobox as _VCombobox,
  VDialog as _VDialog,
  VDivider as _VDivider,
  VIcon as _VIcon,
  VList as _VList,
  VListItem as _VListItem,
  VListSubheader as _VListSubheader,
  VMenu as _VMenu,
  VProgressLinear as _VProgressLinear,
  VSpacer as _VSpacer,
  VSelect as _VSelect,
  VSwitch as _VSwitch,
  VTab as _VTab,
  VTabs as _VTabs,
  VTextField as _VTextField,
  VWindow as _VWindow,
  VWindowItem as _VWindowItem,
} from "vuetify/components";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyComponent = DefineComponent<Record<string, any>, Record<string, any>, any>;

export const VBtn = _VBtn as unknown as AnyComponent;
export const VSelect = _VSelect as unknown as AnyComponent;
export const VBtnToggle = _VBtnToggle as unknown as AnyComponent;
export const VCard = _VCard as unknown as AnyComponent;
export const VCardActions = _VCardActions as unknown as AnyComponent;
export const VCardText = _VCardText as unknown as AnyComponent;
export const VCardTitle = _VCardTitle as unknown as AnyComponent;
export const VCheckbox = _VCheckbox as unknown as AnyComponent;
export const VChip = _VChip as unknown as AnyComponent;
export const VCombobox = _VCombobox as unknown as AnyComponent;
export const VDialog = _VDialog as unknown as AnyComponent;
export const VDivider = _VDivider as unknown as AnyComponent;
export const VIcon = _VIcon as unknown as AnyComponent;
export const VList = _VList as unknown as AnyComponent;
export const VListItem = _VListItem as unknown as AnyComponent;
export const VListSubheader = _VListSubheader as unknown as AnyComponent;
export const VMenu = _VMenu as unknown as AnyComponent;
export const VProgressLinear = _VProgressLinear as unknown as AnyComponent;
export const VSpacer = _VSpacer as unknown as AnyComponent;
export const VSwitch = _VSwitch as unknown as AnyComponent;
export const VTab = _VTab as unknown as AnyComponent;
export const VTabs = _VTabs as unknown as AnyComponent;
export const VTextField = _VTextField as unknown as AnyComponent;
export const VWindow = _VWindow as unknown as AnyComponent;
export const VWindowItem = _VWindowItem as unknown as AnyComponent;
