// mmu.yapl - WUT-4 MMU management library
//
// Functions for privileged (kernel) code to manage the MMU.
// Compile to a relocatable object and link with programs that need it:
//
//   ya -c -o lib/mmu.wo lib/mmu.yapl
//
// MMU Special Register Layout:
//   SPR 32-47: User Code MMU  (16 entries, one per 4K virtual page)
//   SPR 48-63: User Data MMU
//   SPR 64-79: Kernel Code MMU
//   SPR 80-95: Kernel Data MMU
//
// MMU Entry Format (16-bit):
//   Bits 15:4  = physical page number (12 bits, 0..4095)
//   Bits  3:2  = reserved (zero)
//   Bits  1:0  = permissions (PP): 00=all, 01=RO/XO, 10=reserved, 11=invalid
//
// MapPage details argument bit assignments:
//   Bit  0     = ISKERNEL: 1 to update kernel MMU, 0 for current user context
//   Bit  1     = ISCODE:   1 for code space entry, 0 for data space entry
//   Bits 13:12 = permission bits placed verbatim into the MMU entry

// Permission constants (OR into details bits 13:12)
const uint16 PERM_ALL     = 0x0000;   // 00: all permissions (read/write/execute)
const uint16 PERM_RO      = 0x1000;   // 01: read-only (data) / execute-only (code)
const uint16 PERM_INVALID = 0x3000;   // 11: invalid; any access causes page fault

// Detail flag constants (OR into details bits 1:0)
const uint16 ISKERNEL = 1;   // bit 0: update kernel address space
const uint16 ISCODE   = 2;   // bit 1: create code (vs. data) MMU entry

// MapPage: install one MMU entry mapping virtual page virt to physical page phys.
//
// Must be called from kernel (privileged) mode.
//
// details: combination of ISKERNEL, ISCODE, and PERM_* constants OR'd together
// virt:    virtual page number (0..15; 16 pages of 4K cover the 64K address space)
// phys:    physical page number (0..4095; 4096 pages of 4K cover 16MB physical)
//
// Example: map virtual page 0 of user data space to physical page 7, read-only:
//   MapPage(PERM_RO, 0, 7);
//
// Example: map virtual page 2 of kernel code space to physical page 100:
//   MapPage(ISKERNEL | ISCODE, 2, 100);

extern func void MapPage(uint16 details, uint16 virt, uint16 phys);
