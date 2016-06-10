// +build linux

package system_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/system"
	"github.com/pivotal-cf/pcfdev-cli/system/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("system", func() {
	cpuinfo := `
processor    : 0
vendor_id    : GenuineIntel
cpu family    : 6
model        : 60
model name    : Intel(R) Core(TM) i5-4690 CPU @ 3.50GHz
stepping    : 3
cpu MHz        : 3491.916
cache size    : 6144 KB
physical id    : 0
siblings    : 5
core id        : 0
cpu cores    : 5
apicid        : 0
initial apicid    : 0
fpu        : yes
fpu_exception    : yes
cpuid level    : 13
wp        : yes
flags        : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology nonstop_tsc pni pclmulqdq ssse3 cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx rdrand hypervisor lahf_lm abm
bugs        :
bogomips    : 6983.83
clflush size    : 64
cache_alignment    : 64
address sizes    : 39 bits physical, 48 bits virtual
power management:

processor    : 1
vendor_id    : GenuineIntel
cpu family    : 6
model        : 60
model name    : Intel(R) Core(TM) i5-4690 CPU @ 3.50GHz
stepping    : 3
cpu MHz        : 3491.916
cache size    : 6144 KB
physical id    : 0
siblings    : 5
core id        : 1
cpu cores    : 5
apicid        : 1
initial apicid    : 1
fpu        : yes
fpu_exception    : yes
cpuid level    : 13
wp        : yes
flags        : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology nonstop_tsc pni pclmulqdq ssse3 cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx rdrand hypervisor lahf_lm abm
bugs        :
bogomips    : 6983.83
clflush size    : 64
cache_alignment    : 64
address sizes    : 39 bits physical, 48 bits virtual
power management:

processor    : 2
vendor_id    : GenuineIntel
cpu family    : 6
model        : 60
model name    : Intel(R) Core(TM) i5-4690 CPU @ 3.50GHz
stepping    : 3
cpu MHz        : 3491.916
cache size    : 6144 KB
physical id    : 0
siblings    : 5
core id        : 0
cpu cores    : 5
apicid        : 2
initial apicid    : 2
fpu        : yes
fpu_exception    : yes
cpuid level    : 13
wp        : yes
flags        : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology nonstop_tsc pni pclmulqdq ssse3 cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx rdrand hypervisor lahf_lm abm
bugs        :
bogomips    : 6983.83
clflush size    : 64
cache_alignment    : 64
address sizes    : 39 bits physical, 48 bits virtual
power management:

processor    : 3
vendor_id    : GenuineIntel
cpu family    : 6
model        : 60
model name    : Intel(R) Core(TM) i5-4690 CPU @ 3.50GHz
stepping    : 3
cpu MHz        : 3491.916
cache size    : 6144 KB
physical id    : 0
siblings    : 5
core id        : 1
cpu cores    : 5
apicid        : 3
initial apicid    : 3
fpu        : yes
fpu_exception    : yes
cpuid level    : 13
wp        : yes
flags        : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology nonstop_tsc pni pclmulqdq ssse3 cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx rdrand hypervisor lahf_lm abm
bugs        :
bogomips    : 6983.83
clflush size    : 64
cache_alignment    : 64
address sizes    : 39 bits physical, 48 bits virtual
power management:

processor    : 4
vendor_id    : GenuineIntel
cpu family    : 6
model        : 60
model name    : Intel(R) Core(TM) i5-4690 CPU @ 3.50GHz
stepping    : 3
cpu MHz        : 3491.916
cache size    : 6144 KB
physical id    : 1
siblings    : 5
core id        : 0
cpu cores    : 5
apicid        : 0
initial apicid    : 0
fpu        : yes
fpu_exception    : yes
cpuid level    : 13
wp        : yes
flags        : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology nonstop_tsc pni pclmulqdq ssse3 cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx rdrand hypervisor lahf_lm abm
bugs        :
bogomips    : 6983.83
clflush size    : 64
cache_alignment    : 64
address sizes    : 39 bits physical, 48 bits virtual
power management:

processor    : 5
vendor_id    : GenuineIntel
cpu family    : 6
model        : 60
model name    : Intel(R) Core(TM) i5-4690 CPU @ 3.50GHz
stepping    : 3
cpu MHz        : 3491.916
cache size    : 6144 KB
physical id    : 1
siblings    : 5
core id        : 1
cpu cores    : 5
apicid        : 1
initial apicid    : 1
fpu        : yes
fpu_exception    : yes
cpuid level    : 13
wp        : yes
flags        : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology nonstop_tsc pni pclmulqdq ssse3 cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx rdrand hypervisor lahf_lm abm
bugs        :
bogomips    : 6983.83
clflush size    : 64
cache_alignment    : 64
address sizes    : 39 bits physical, 48 bits virtual
power management:

processor    : 6
vendor_id    : GenuineIntel
cpu family    : 6
model        : 60
model name    : Intel(R) Core(TM) i5-4690 CPU @ 3.50GHz
stepping    : 3
cpu MHz        : 3491.916
cache size    : 6144 KB
physical id    : 1
siblings    : 5
core id        : 0
cpu cores    : 5
apicid        : 2
initial apicid    : 2
fpu        : yes
fpu_exception    : yes
cpuid level    : 13
wp        : yes
flags        : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology nonstop_tsc pni pclmulqdq ssse3 cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx rdrand hypervisor lahf_lm abm
bugs        :
bogomips    : 6983.83
clflush size    : 64
cache_alignment    : 64
address sizes    : 39 bits physical, 48 bits virtual
power management:

processor    : 7
vendor_id    : GenuineIntel
cpu family    : 6
model        : 60
model name    : Intel(R) Core(TM) i5-4690 CPU @ 3.50GHz
stepping    : 3
cpu MHz        : 3491.916
cache size    : 6144 KB
physical id    : 1
siblings    : 5
core id        : 1
cpu cores    : 5
apicid        : 3
initial apicid    : 3
fpu        : yes
fpu_exception    : yes
cpuid level    : 13
wp        : yes
flags        : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology nonstop_tsc pni pclmulqdq ssse3 cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx rdrand hypervisor lahf_lm abm
bugs        :
bogomips    : 6983.83
clflush size    : 64
cache_alignment    : 64
address sizes    : 39 bits physical, 48 bits virtual
power management:
`
	var (
		mockCtrl *gomock.Controller
		mockFS   *mocks.MockFS
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should return the number of physical cores", func() {
		sys := system.System{
			FS: mockFS,
		}
		mockFS.EXPECT().Read("/proc/cpuinfo").Return([]byte(cpuinfo), nil)
		cores, err := sys.PhysicalCores()
		Expect(err).NotTo(HaveOccurred())
		Expect(cores).To(Equal(4))
	})

	Context("when it cannot read cpuinfo", func() {
		It("should return an error", func() {
			sys := system.System{
				FS: mockFS,
			}
			mockFS.EXPECT().Read("/proc/cpuinfo").Return(nil, errors.New("some-error"))
			_, err := sys.PhysicalCores()
			Expect(err).To(MatchError("some-error"))
		})

	})
})
